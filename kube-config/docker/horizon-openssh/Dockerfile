FROM ubuntu:15.10

RUN apt-get update && \
    apt-get dist-upgrade -y && \
    apt-get install -y ssh rsync && \
    rm -rf /etc/ssh/ssh_host_* /var/lib/apt/lists/* && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    yes '' | adduser --disabled-password horizon

ADD authorized_keys /home/horizon/.ssh/authorized_keys
RUN chown -R horizon:horizon /home/horizon/.ssh

COPY run.sh /root/run.sh
COPY post-deploy.sh /home/horizon/post-deploy.sh
RUN chown horizon:horizon /home/horizon/post-deploy.sh

EXPOSE 22
CMD ["/root/run.sh"]
