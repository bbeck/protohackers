.DEFAULT_GOAL := help

# Canonicalize the PROBLEM parameter so it's suitable to use in directory
# names.  This will ensure that it has the correct length with leading zeroes.
override PROBLEM := $(shell printf '%02d' $$((10\#$(PROBLEM))))

## run the solution for the specified PROBLEM
.PHONY: run
run:
	@go run cmd/problem-$(PROBLEM)/*.go

## watch for changes and rerun the solution for the specified PROBLEM
.PHONY: watch
watch:
	@find                                                          \
	    internal                                                   \
	    cmd/problem-$(PROBLEM)                                     \
	  -type f                                                    | \
	 entr -c -d -r make -s run PROBLEM=$(PROBLEM)

## display this help message
.PHONY: help
help:
	@awk '                                                      \
	  BEGIN {                                                   \
	    printf "Usage:\n"                                       \
	  }                                                         \
	                                                            \
	  /^##@/ {                                                  \
	    printf "\n\033[1m%s:\033[0m\n", substr($$0, 5)          \
	  }                                                         \
	                                                            \
	  /^##([^@]|$$)/ && $$2 != "" {                             \
	    $$1 = "";                                               \
	    if (message == null) {                                  \
	      message = $$0;                                        \
	    } else {                                                \
	      message = message "\n           " $$0;                \
	    }                                                       \
	  }                                                         \
	                                                            \
	  /^[a-zA-Z_-]+:/ && message != null {                      \
	    target = substr($$1, 0, length($$1)-1);                 \
	    printf "  \033[36m%-8s\033[0m %s\n", target, message;   \
	    message = null;                                         \
	  }                                                         \
	' $(MAKEFILE_LIST)
