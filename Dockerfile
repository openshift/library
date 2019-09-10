# Dockerfile used to verify openshift/library via ci-operator
FROM docker.io/centos/python-36-centos7:latest

USER root
RUN yum install -y git
USER default

COPY . ${HOME}

RUN pip install -U pip && \
    pip install -r ${HOME}/requirements.txt

CMD ["make", "verify"]
