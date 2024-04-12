.PHONY: check-env download test test-headed-chrome test-headed-firefox clean

download:
	npm install && npx playwright install

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
