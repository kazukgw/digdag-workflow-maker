package main

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mholt/archiver"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/pressly/chi/render"
)

func main() {
	mux := chi.NewRouter()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	mux.Use(middleware.CloseNotify)
	mux.Use(middleware.Timeout(60 * time.Second))

	cwd, _ := os.Getwd()
	staticdir := http.Dir(cwd + "/static")
	staticfs := http.FileServer(staticdir)
	mux.Handle("/static/*", http.StripPrefix("/static/", staticfs))

	appdir := http.Dir(cwd + "/app")
	appfs := http.FileServer(appdir)
	mux.Handle("/app/*", http.StripPrefix("/app/", appfs))

	mux.Post("/bqworkflow", bqworkflowPutHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

type WorkflowJson struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ScheduleType string `json:"scheduleType"`
	Schedule     string `json:"schedule"`
	TasksJson    `json:"tasks"`
}

type TasksJson []TaskJson

type TaskJson struct {
	Name              string `json:"name"`
	Destination       string `json:"destination"`
	CreateDisposition string `json:"createDisposition"`
	WriteDisposition  string `json:"writeDisposition"`
	SQL               string `json:"sql"`
}

func bqworkflowPutHandler(w http.ResponseWriter, r *http.Request) {
	var res struct {
		result string
	}

	var wf WorkflowJson
	if err := render.Bind(r.Body, &wf); err != nil {
		w.WriteHeader(500)
		render.JSON(w, r, err)
		return
	}

	log.Printf("posted workflow: %#v\n", spew.Sprint(wf))

	bqwf := NewDigdagProjectSaver(
		&wf, "workflows", "credential.json",
		"template.dig", "http://digdag:65432/api/projects",
	)

	if err := bqwf.save(); err != nil {
		log.Printf("err: %#v\n", err.Error())
		w.WriteHeader(500)
		render.JSON(w, r, err)
		return
	}
	res.result = "success"
	render.JSON(w, r, res)
}

type DigdagProjectSaver struct {
	WorkDir         string
	ProjDir         string
	CredFile        string
	TemplateDigFile string
	DigdagServer    string
	*WorkflowJson
}

func NewDigdagProjectSaver(
	wf *WorkflowJson,
	workdir,
	credfile,
	templateDigFile,
	digdagServer string,
) *DigdagProjectSaver {
	bqwf := &DigdagProjectSaver{}
	bqwf.WorkflowJson = wf
	now := time.Now().Format("20060102_150405")
	bqwf.WorkDir = workdir
	bqwf.ProjDir = bqwf.WorkDir + "/" + now + "-" + wf.Name
	bqwf.CredFile = credfile
	bqwf.TemplateDigFile = templateDigFile
	bqwf.DigdagServer = digdagServer
	return bqwf
}

func (bqwf *DigdagProjectSaver) save() error {
	log.Println("create project dir")
	if err := os.Mkdir(bqwf.ProjDir, 0755); err != nil {
		return err
	}

	log.Println("copy credential to project dir")
	if err := fileCopy(
		bqwf.CredFile,
		bqwf.ProjDir+"/"+bqwf.CredFile,
	); err != nil {
		return err
	}

	log.Println("create dig file")
	if err := bqwf.createDigFile(); err != nil {
		return err
	}

	log.Println("create sql file")
	if err := bqwf.createSQLFile(); err != nil {
		return err
	}

	log.Println("gzip project")
	archiveName := "digdag-" + bqwf.WorkflowJson.Name + ".tar.gz"
	files, _ := filepath.Glob(bqwf.ProjDir + "/*")
	if err := archiver.TarGz.Make(
		archiveName,
		files,
	); err != nil {
		return err
	}

	log.Println("push to digdag server")
	resp, err := bqwf.push(archiveName)
	if err != nil {
		return err
	}

	log.Println("recv response")
	if resp.StatusCode != 200 {
		dat, _ := ioutil.ReadAll(resp.Body)
		log.Println(string(dat))
		return errors.New("status code is not 200: " + string(dat))
	}

	return nil
}

func (bqwf *DigdagProjectSaver) createDigFile() error {
	tmpl, err := template.ParseFiles(bqwf.TemplateDigFile)
	if err != nil {
		return err
	}
	digfile := bqwf.ProjDir + "/" + bqwf.WorkflowJson.Name + ".dig"
	log.Printf("digfile: %#v\n", digfile)
	f, err := os.OpenFile(
		digfile,
		os.O_WRONLY|os.O_CREATE,
		0755,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, bqwf)
}

func (bqwf *DigdagProjectSaver) createSQLFile() error {
	for _, t := range bqwf.WorkflowJson.TasksJson {
		err := ioutil.WriteFile(
			bqwf.ProjDir+"/"+t.Name+".sql",
			[]byte(t.SQL),
			0755,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bqwf *DigdagProjectSaver) push(archiveName string) (*http.Response, error) {
	endpoint := bqwf.DigdagServer + "?project=" +
		bqwf.WorkflowJson.Name + "&revision=" + time.Now().Format("20060102_150405")
	req, err := newFileUploadRequest(
		endpoint,
		"file",
		archiveName,
	)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}

func fileCopy(source string, dest string) (err error) {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	if err == nil {
		si, err := os.Stat(source)
		if err == nil {
			err = os.Chmod(dest, si.Mode())
		}
	}
	return err

}

func newFileUploadRequest(
	uri string,
	paramName,
	path string,
) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	// writer := multipart.NewWriter(body)
	// part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	_, err = io.Copy(body, file)
	if err != nil {
		return nil, err
	}
	// _, err = io.Copy(part, file)

	// for key, val := range params {
	// _ = writer.WriteField(key, val)
	// }

	req, err := http.NewRequest("PUT", uri, body)
	req.Header.Set("Content-Type", "application/gzip")
	return req, err
}
