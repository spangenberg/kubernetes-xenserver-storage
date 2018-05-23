all build:
	make -C driver
	make -C provisioner
.PHONY: all build

container:
	make -C driver container
	make -C provisioner container
.PHONY: container

push:
	make -C driver push
	make -C provisioner push
.PHONY: push

clean:
	make -C driver clean
	make -C provisioner clean
.PHONY: clean
