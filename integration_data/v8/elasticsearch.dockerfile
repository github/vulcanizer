FROM docker.elastic.co/elasticsearch/elasticsearch:8.13.0

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
