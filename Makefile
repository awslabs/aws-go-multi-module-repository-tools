RELEASE_MANIFEST_FILE ?=
RELEASE_CHGLOG_DESC_FILE ?=

#####################
#  Release Process  #
#####################
.PHONY: preview-release pre-release-validation release

preview-release:
	go run ./cmd/calculaterelease

pre-release-validation:
	@if [[ -z "${RELEASE_MANIFEST_FILE}" ]]; then \
      		echo "RELEASE_MANIFEST_FILE is required to specify the file to write the release manifest" && false; \
      	fi
	@if [[ -z "${RELEASE_CHGLOG_DESC_FILE}" ]]; then \
		echo "RELEASE_CHGLOG_DESC_FILE is required to specify the file to write the release notes" && false; \
	fi

release: pre-release-validation
	go run ./cmd/calculaterelease -o ${RELEASE_MANIFEST_FILE}
	go run ./cmd/updaterequires -release ${RELEASE_MANIFEST_FILE}
	go run ./cmd/updatemodulemeta -release ${RELEASE_MANIFEST_FILE}
	go run ./cmd/generatechangelog -release ${RELEASE_MANIFEST_FILE} -o ${RELEASE_CHGLOG_DESC_FILE}
	go run ./cmd/changelog rm -all
	go run ./cmd/tagrelease -release ${RELEASE_MANIFEST_FILE}

install-tools:
	go install github.com/plexsystems/pacmod@v0.4.0

build:
	go build ./...

update-mod-package: clean-mod-package package-mod

clean-mod-package:
	@if [[ -z "${MOD_PKG_ROOT}" ]]; then \
		echo "MOD_PKG_ROOT is required to output mod package to" && false; \
	fi
	rm -rf ${MOD_PKG_ROOT}/github.com/awslabs/aws-go-multi-module-repository-tools/*

package-mod: build
	@if [[ -z "${MOD_PKG_ROOT}" ]]; then \
		echo "MOD_PKG_ROOT is required to output mod package to" && false; \
	fi
	@if [[ -z "${REPOTOOLS_VERSION}" ]]; then \
		echo "REPOTOOLS_VERSION is required to specify version to package" && false; \
	fi
	pacmod pack ${REPOTOOLS_VERSION} .
	mkdir -p ${MOD_PKG_ROOT}/github.com/awslabs/aws-go-multi-module-repository-tools/${REPOTOOLS_VERSION}
	cp go.mod ${MOD_PKG_ROOT}/github.com/awslabs/aws-go-multi-module-repository-tools/${REPOTOOLS_VERSION}/go.mod
	mv ${REPOTOOLS_VERSION}.info ${MOD_PKG_ROOT}/github.com/awslabs/aws-go-multi-module-repository-tools/${REPOTOOLS_VERSION}/${REPOTOOLS_VERSION}.info
	mv ${REPOTOOLS_VERSION}.zip ${MOD_PKG_ROOT}/github.com/awslabs/aws-go-multi-module-repository-tools/${REPOTOOLS_VERSION}/source.zip

