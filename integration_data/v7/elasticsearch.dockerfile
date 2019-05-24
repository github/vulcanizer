FROM docker.elastic.co/elasticsearch/elasticsearch:7.1.0

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
