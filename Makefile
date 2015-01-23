IMAGE_UUID := $(shell uuidgen)
CONTAINER_UUID := $(shell uuidgen)

default: docker

docker:
	mkdir -p ./build
	docker build --no-cache -t "$(IMAGE_UUID)" .
	docker run --name="$(CONTAINER_UUID)" "$(IMAGE_UUID)" /bin/bash
	docker cp $(CONTAINER_UUID):/usr/local/bin/onmetal-rescue-agent ./build
	docker cp $(CONTAINER_UUID):/usr/local/bin/finalize_rescue.bash ./build
	docker rm $(CONTAINER_UUID)

clean:
	rm -rf ./build
