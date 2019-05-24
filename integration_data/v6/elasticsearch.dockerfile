FROM docker.elastic.co/elasticsearch/elasticsearch:6.8.0

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
