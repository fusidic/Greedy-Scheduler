FROM debian:stretch-slim

WORKDIR /

COPY greedy-scheduler /usr/local/bin

CMD ["greedy-scheduler"]