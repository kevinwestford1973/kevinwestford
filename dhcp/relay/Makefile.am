BINDLIBDIR = @BINDDIR@/lib

AM_CPPFLAGS = -DLOCALSTATEDIR='"@localstatedir@"'

sbin_PROGRAMS = dhcrelay
dhcrelay_SOURCES = dhcrelay.c ccmts_kafka.c ccmts_infra_kafka.c
dhcrelay_LDADD = ../common/libdhcp.a ../omapip/libomapi.a \
		 $(BINDLIBDIR)/libirs.a $(BINDLIBDIR)/libdns.a \
		 $(BINDLIBDIR)/libisccfg.a $(BINDLIBDIR)/libisc.a \
		 ../../infra/libs/librdkafka.so \
		 ../../infra/libs/libsasl2.so

man_MANS = dhcrelay.8
EXTRA_DIST = $(man_MANS)

