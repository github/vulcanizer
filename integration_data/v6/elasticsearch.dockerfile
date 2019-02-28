FROM docker.elastic.co/elasticsearch/elasticsearch:6.6.1

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
