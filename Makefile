#
# Targets (see each target for more information):
#   verify: Run verify.
#   import: Run the import script.
#

# Run core verification.
#
# Example:
#   make verify
verify:
	hack/verify-python.sh
	hack/verify-yaml.sh
	hack/verify-pullrequest.sh
.PHONY: verify

# Run the import script.
#
# Example:
#   make import
#   make import TAGS=online-starter,online-professional
import:
	python import_content.py -t ${TAGS}
.PHONY: import
