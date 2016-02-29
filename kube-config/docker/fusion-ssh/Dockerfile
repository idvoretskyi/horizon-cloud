FROM ubuntu:15.10

RUN apt-get update && \
    apt-get dist-upgrade -y && \
    apt-get install -y ssh rsync && \
    rm -rf /etc/ssh/ssh_host_* /var/lib/apt/lists/* && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    yes '' | adduser --disabled-password fusion

ADD authorized_keys /home/fusion/.ssh/authorized_keys
RUN chown -R fusion:fusion /home/fusion/.ssh

COPY run.sh /root/run.sh

EXPOSE 22
CMD ["/root/run.sh"]
