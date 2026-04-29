PREFIX ?= $(HOME)/.local

.PHONY: install uninstall list

install:
	@mkdir -p $(PREFIX)/bin
	@for f in bin/*; do \
		ln -sf $(CURDIR)/$$f $(PREFIX)/bin/$$(basename $$f); \
		echo "  linked $(PREFIX)/bin/$$(basename $$f)"; \
	done

uninstall:
	@for f in bin/*; do \
		rm -f $(PREFIX)/bin/$$(basename $$f); \
		echo "  removed $(PREFIX)/bin/$$(basename $$f)"; \
	done

list:
	@echo "Available tools:"
	@for f in bin/*; do echo "  $$(basename $$f)"; done
