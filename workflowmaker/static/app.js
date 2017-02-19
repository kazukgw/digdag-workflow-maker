Vue.use(VueMaterial)

var App = new Vue({
  el: '#app',
  data: {
    saved: false,
    workflow: {
      name: "",
      description: "",
      scheduleType: "",
      schedule: "",
      tasks: [{
        name: "",
        destination: "",
        createDisposition: "",
        wirteDisposition: "",
        sql: ""
      }]
    },
    validator: {
      name: false,
      description: false,
      sheduleType: false,
      schedule: false,
      tasks: [{
        name: false,
        destination: false,
        createDisposition: false,
        writeDisposition: false,
        sql: false
      }],
    },
    saveStatus: {
      saving: true,
      result: 0
    }
  },
  methods: {
    addTask: function(){
      var newtask = {
        name: "",
        destination: "",
        createDisposition: "",
        wirteDisposition: "",
        sql: ""
      };
      this.workflow.tasks.push(newtask);

      var newtaskValidator = {
        name: false,
        destination: false,
        createDisposition: false,
        writeDisposition: false,
        sql: false
      }
      this.validator.tasks.push(newtaskValidator);
    },
    deleteTaskWithIndex: function(taskIndex) {
      this.workflow.tasks.splice(taskIndex, 1)
    },
    isValid: function(validator) {
      for (key in validator) {
        if (key === "tasks") {
          tasks = validator[tasks]
          for (var i=0, l=tasks.length; i < l; i++) {
            if(!tasks[i]) {
              return false;
            }
          }
        }
        if (!validator[key]) {
          return false;
        }
      }
    },
    saveWorkflow: function() {
      this.saveStatus.saving = true;
      console.log("post wf:", this.workflow);
      // if (!this.isValid(this.validator)) {
      //   this.$refs.notValidDialog.open();
      //   return;
      // }
      var self = this;
      if (this.saved) {
        this.$refs.messageDialog.open();
        return;
      }
      this.$refs.saveDialog.open();
      setTimeout(function(){
        axios.post("/bqworkflow", self.workflow)
          .then(function(response){
            console.log("resp:", response);
            self.saveStatus.result = "Success !";
            self.saveStatus.saving = false
            self.saved = true;
          })
          .catch(function(err){
            console.log(err);
            self.saveStatus.saving = false
            self.saveStatus.result = "Fault !";
          })
      }, 600);
    }
  }
});
