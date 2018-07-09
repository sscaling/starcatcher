Crontab + setup
---------------

```
mkdir -p /opt/code
docker run --rm -v /opt/code:/repo sscaling/robotgit sh -c "git clone git@github.com:sscaling/project-stats.git"

# Add starcatcher script with chmod +x to /opt/code/project-stats
vi /etc/config/crontab
* 5 * * * /opt/code/project-stats/starcatcher.sh

crontab /etc/config/crontab && /etc/init.d/crond.sh restart
```

starcatcher.sh
--------------

```
#!/bin/sh

docker run --rm -v /opt/code/project-stats/data:/data sscaling/starcatcher "https://api.github.com/repos/wurstmeister/kafka-docker" "https://hub.docker.com/v2/repositories/wurstmeister/kafka/" /data/kafka.csv /data/kafka.png
docker run --rm -v /opt/code/project-stats/data:/data sscaling/starcatcher "https://api.github.com/repos/sscaling/docker-jmx-prometheus-exporter" "https://hub.docker.com/v2/repositories/sscaling/jmx-prometheus-exporter/" /data/jmx-exporter.csv /data/jmx-exporter.png

docker run --rm -v $PWD:/repo sscaling/robotgit sh -c "git add -u"
docker run --rm -v $PWD:/repo sscaling/robotgit sh -c 'git commit -m "$(date -Iseconds) data"'
docker run --rm -v $PWD:/repo sscaling/robotgit sh -c 'git push'
```
