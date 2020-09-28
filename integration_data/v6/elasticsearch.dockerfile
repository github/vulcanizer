FROM docker.elastic.co/elasticsearch/elasticsearch:6.8.12

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
