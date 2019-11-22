FROM docker.elastic.co/elasticsearch/elasticsearch:5.6.15

USER root

RUN mkdir /backups && chown elasticsearch:elasticsearch /backups

RUN sysctl -w vm.max_map_count=262144

USER elasticsearch
