# Dockerfile used to verify openshift/library via ci-operator
FROM registry.svc.ci.openshift.org/openshift/release:golang-1.13 as builder
WORKDIR /go/src/github.com/openshift/library
COPY . .
RUN make verify-gofmt
RUN make build

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
RUN yum install -y git make
COPY --from=builder /go/src/github.com/openshift/library /go/src/github.com/openshift/library
RUN chmod 777 /go/src/github.com/openshift/library
WORKDIR /go/src/github.com/openshift/library
ENTRYPOINT []
CMD ["make", "verify-pullrequest"]
