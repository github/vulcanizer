FROM docker.elastic.co/elasticsearch/elasticsearch:6.5.2

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
