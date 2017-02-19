FROM openjdk:8

ARG DIGDAG_VERSION=latest
ARG EMBULK_VERSION=latest

RUN useradd digdag -m -s /bin/bash \
  && apt-key adv \
    --keyserver hkp://p80.pool.sks-keyservers.net:80 \
    --recv-keys 58118E89F3A912897C070ADBF76221572C52609D \
  && apt-get update \
  && apt-get install apt-transport-https ca-certificates gnupg2 -y \
  && apt-get install \
    git openssh-client jruby tzdata connect-proxy gfortran libffi-dev \
    python-setuptools python-dev zlibc freetds-dev python-mysqldb openssl \
    mysql-client libssl-dev libxml2-dev libxslt1-dev zlib1g-dev connect-proxy -y \
  && easy_install pip \
  && curl --create-dirs -o /usr/local/bin/digdag -L -k "https://dl.digdag.io/digdag-$DIGDAG_VERSION" \
  && chmod +x /usr/local/bin/digdag \
  && curl --create-dirs -o /usr/local/bin/embulk -L -k "https://dl.embulk.org/embulk-$EMBULK_VERSION.jar" \
  && chmod +x /usr/local/bin/embulk \
  && embulk gem install \
    embulk-input-s3 \
    embulk-output-s3 \
    embulk-input-mysql \
    embulk-output-mysql \
    embulk-input-postgresql \
    embulk-output-postgresql \
    embulk-input-gcs \
    embulk-output-gcs \
    embulk-input-jdbc \
    embulk-output-jdbc \
    embulk-input-sqlserver \
    embulk-put-sqlserver \
    embulk-output-bigquery \
    # filter
    embulk-filter-column \
    embulk-filter-row \
    embulk-filter-kuromoji \
    embulk-filter-insert \
    embulk-filter-hash \
    # parser
    embulk-parser-jsonl \
    # formatter
    embulk-formatter-jsonl \
  && apt-get clean \
  && apt-get autoremove \
  && dpkg -l 'linux-*' \
    | sed '/^ii/!d;/'"$(uname -r | sed "s/\(.*\)-\([^0-9]\+\)/\1/")"'/d;s/^[^ ]* [^ ]* \([^ ]*\).*/\1/;/[0-9]/!d' \
    | xargs apt-get -y purge

USER digdag
WORKDIR /home/digdag
EXPOSE 65432
