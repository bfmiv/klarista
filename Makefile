export SHELL = /bin/bash

export GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null)
export GIT_COMMIT ?= $(shell git rev-parse --short=7 HEAD 2> /dev/null)
export GIT_DIRTY  ?= $(shell [[ `git status --short --untracked-files=no` ]] && echo 1)

export DOCKER_BRANCH_TAG = '$(GIT_BRANCH)$(if $(GIT_DIRTY),--DIRTY)'
export DOCKER_COMMIT_TAG = '$(GIT_COMMIT)$(if $(GIT_DIRTY),--DIRTY)'
export DOCKER_IMAGE      = "bernardmcmanus/klarista"

export KLARISTA_CLI_VERSION               = '0.8.1'
export KLARISTA_CLI_VERSION_TAG           = '$(KLARISTA_CLI_VERSION)$(if $(GIT_DIRTY),--DIRTY)'
export KLARISTA_CLI_VERSION_ALIAS_TAG = '$(shell awk 'BEGIN { FS = "." } ; { print $$1 "." $$2 }' <<< '$(KLARISTA_CLI_VERSION)')$(if $(GIT_DIRTY),--DIRTY)'

define each_release_tag
set -euo pipefail; \
for t in $(DOCKER_BRANCH_TAG)-release $(DOCKER_COMMIT_TAG)-release $(KLARISTA_CLI_VERSION_TAG) $(KLARISTA_CLI_VERSION_ALIAS_TAG) latest; do \
	function repotag { echo $(DOCKER_IMAGE):$$t; }; \
	$1; \
done
endef

.DEFAULT: all

all: dev test release

.PHONY: dev
dev:
	@docker build \
		-t $(DOCKER_IMAGE):$(DOCKER_BRANCH_TAG)-dev \
		-t $(DOCKER_IMAGE):$(DOCKER_COMMIT_TAG)-dev \
		--target dev \
		.

.PHONY: test
test:
	@docker run \
		--rm \
		$(DOCKER_IMAGE):$(DOCKER_COMMIT_TAG)-dev \
		bash -c 'echo "FIXME: RUN TESTS!"'

.PHONY: release
release:
	@docker build \
		-t $(DOCKER_IMAGE):$(DOCKER_BRANCH_TAG)-release \
		-t $(DOCKER_IMAGE):$(DOCKER_COMMIT_TAG)-release \
		-t $(DOCKER_IMAGE):$(KLARISTA_CLI_VERSION_TAG) \
		-t $(DOCKER_IMAGE):$(KLARISTA_CLI_VERSION_ALIAS_TAG) \
		-t $(DOCKER_IMAGE):latest \
		--build-arg KLARISTA_CLI_VERSION=$(KLARISTA_CLI_VERSION) \
		.

.PHONY: push
push:
	@$(call each_release_tag, docker push `repotag`)

.PHONY: install
install: dev release
	@docker run \
		--rm \
		$(DOCKER_IMAGE):$(DOCKER_COMMIT_TAG)-release \
		install | bash
