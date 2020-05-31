FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/sample-mysql-operator \
    USER_UID=1001 \
    USER_NAME=sample-mysql-operator

# install operator binary
COPY build/_output/bin/sample-mysql-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
