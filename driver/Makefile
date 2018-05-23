IMAGE = spangenberg/xenserver-driver:latest

all build container quick-container:
	docker build -t $(IMAGE) .
.PHONY: all build container quick-container

push: container
	docker push $(IMAGE)
.PHONY: push
