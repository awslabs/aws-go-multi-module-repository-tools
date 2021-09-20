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

