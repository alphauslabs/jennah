.PHONY: build gw-docker-build gw-docker-run gw-docker-push clean

PROJECT_ID = labs-169405
IMAGE_NAME = jennah-gateway
IMAGE_TAG = latest

generate:
	buf generate --exclude-path vendor/

gw-docker-build:
	docker build -f Dockerfile.gateway -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) gcr.io/$(PROJECT_ID)/$(IMAGE_NAME):$(IMAGE_TAG)

gw-docker-run:
	docker run --rm -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

gw-docker-push:
	docker push gcr.io/$(PROJECT_ID)/$(IMAGE_NAME):$(IMAGE_TAG)

clean:
	rm -rf bin/
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true
	docker rmi gcr.io/$(PROJECT_ID)/$(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true