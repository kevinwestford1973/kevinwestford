FROM alpine:3.5

MAINTAINER ccmts team

LABEL Description="This image is used to run DHCP relay micro services" Vendor="Cisco Systems" Version="1.0.0"

RUN mkdir -p /root/cmts-l3-dhcprelay/bin

WORKDIR /root/cmts-l3-dhcprelay/bin

ADD bin /root/cmts-l3-dhcprelay/bin

ADD startup.sh /root/cmts-l3-dhcprelay/bin

ADD bin/libicpi.so /lib

RUN chmod +x /root/cmts-l3-dhcprelay/bin/startup.sh

#RUN /usr/glibc-compat/sbin/ldconfig /lib /usr/glibc/usr/lib

ENTRYPOINT ["./startup.sh"]

#EXPOSE 9002/udp
#EXPOSE 9003/udp


