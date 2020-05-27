FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/aws-nlb-helper-operator \
    USER_UID=1001 \
    USER_NAME=aws-nlb-helper-operator

# install operator binary
COPY build/_output/bin/aws-nlb-helper-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
