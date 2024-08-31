# Add
# export JAVA_HOME=/usr/lib/jvm/java-21
# before any building goal

.PHONY: check-env download test test-headed-chrome test-headed-firefox clean install_node_deps patch_playwright install_browsers

install_node_deps:
	npm install

install_browsers:
	npx playwright install

patch_playwright:
	sed -e '/utils.wrapInASCIIBox/c\// This line is removed for sake Fedora.' -i ./node_modules/playwright-core/lib/server/registry/dependencies.js

download: install_node_deps patch_playwright install_browsers

check-env:
	echo -n 'node: ' && node --version

test:
	npx playwright test

test-headed-chrome:
	npx playwright test --headed --project=chromium

test-headed-firefox:
	npx playwright test --headed --project=firefox

clean:
	rm -rf test-results
