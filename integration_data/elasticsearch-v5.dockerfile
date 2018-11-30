FROM docker.elastic.co/elasticsearch/elasticsearch:5.6.13

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

USER elasticsearch
