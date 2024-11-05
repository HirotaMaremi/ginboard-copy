FROM mysql:5.7

RUN rm -rf /var/lib/mysql && \
mkdir /var/lib/mysql && \
chown -R mysql /var/lib/mysql && \
chown -R mysql /tmp && \
chgrp -R mysql /var/lib/mysql

ENV TZ Asia/Tokyo