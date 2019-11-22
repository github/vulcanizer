FROM docker.elastic.co/elasticsearch/elasticsearch:7.4.2

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
