FROM ubuntu:latest
LABEL authors="drplx"

ENTRYPOINT ["top", "-b"]