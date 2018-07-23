IMAGE = spangenberg/xenserver-provisioner:latest

all build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/xenserver-provisioner
.PHONY: all build

container: build quick-container
.PHONY: container

quick-container:
	docker build -t $(IMAGE) .
.PHONY: quick-container

push: container
	docker push $(IMAGE)
.PHONY: push

clean:
	rm -f bin/xenserver-provisioner
.PHONY: clean
