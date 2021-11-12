FROM gcr.io/distroless/static:nonroot

COPY build/_output/bin/aws-nlb-helper-operator \
  /usr/local/bin/aws-nlb-helper-operator

ENTRYPOINT ["/usr/local/bin/aws-nlb-helper-operator"]