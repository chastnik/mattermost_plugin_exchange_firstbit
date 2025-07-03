## Builds the webapp, if it exists.
.PHONY: webapp
webapp: webapp/dist

webapp/dist: $(WEBAPP_FILES)
	cd webapp && $(NPM) install --no-save
	cd webapp && $(NPM) run build; 