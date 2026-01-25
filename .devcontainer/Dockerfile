# syntax=docker/dockerfile:1

ARG VARIANT="1.25.6-bookworm"
FROM golang:${VARIANT}

ARG USERNAME="vscode"
ARG USER_UID="1000"
ARG USER_GID="1000"

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash-completion \
        ca-certificates \
        curl \
        git \
        iproute2 \
        less \
        make \
        openssh-client \
        sudo \
        tzdata \
        unzip \
        vim \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid ${USER_GID} ${USERNAME} \
    && useradd --uid ${USER_UID} --gid ${USER_GID} -m ${USERNAME} \
    && echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} \
    && chmod 0440 /etc/sudoers.d/${USERNAME} \
    && chown -R ${USERNAME}:${USERNAME} /go /usr/local/go

ENV GO111MODULE=on

WORKDIR /workspaces/pagerduty-oncall-notifier

USER ${USERNAME}
