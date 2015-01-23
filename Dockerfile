FROM stackbrew/ubuntu:trusty

# The add is before the RUN to ensure we get the latest version of packages
# Docker will cache RUN commands, but because the SHA1 of the dir will be
# different it will not cache this layer
ADD . /tmp/onmetal-rescue-agent

# Install requirements for onmetal-rescue-agent
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends golang && \
    apt-get -y autoremove && \
    apt-get clean

# This will succeed because all the dependencies were installed previously
RUN cd /tmp/onmetal-rescue-agent && \ 
    go build . && \
    cp ./onmetal-rescue-agent /usr/local/bin/ && \
    cp ./lib/finalize_rescue.bash /usr/local/bin/

RUN rm -rf /tmp/onmetal-rescue-agent
RUN rm -rf /var/lib/apt/lists/*

RUN apt-get -y purge golang && \
    apt-get -y autoremove && \
    apt-get clean

CMD [ "/usr/local/bin/onmetal-rescue-agent" ]
