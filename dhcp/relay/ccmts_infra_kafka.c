
/*
 * Cloud CMTS kafka library wrapper
 */

#include <ctype.h>
#include <signal.h>
#include <string.h>
#include <unistd.h>
#include <stdlib.h>
#include <syslog.h>
#include <errno.h>
#include <sys/time.h>
#include "event.h"

#include "../librdkafka/src/rdkafka.h"
#include "ccmts_infra_kafka.h"

static int quiet = 0;

#define TRUE 1
#define FALSE 0

#define KAFKA_DEBUG(format, ...) \
{ \
    fprintf(stdout, format, ##__VA_ARGS__);\
    fflush(stdout); \
}

extern uint64_t get_time_in_usec ( );

static enum {
    OUTPUT_HEXDUMP, OUTPUT_RAW,
} output = OUTPUT_RAW;

static void hexdump(FILE *fp, const char *name, const void *ptr, size_t len) {
    const char *p = (const char *) ptr;
    size_t of = 0;

    if (name)
        fprintf(fp, "%s hexdump (%zd bytes):\n", name, len);

    for (of = 0; of < len; of += 16) {
        char hexen[16 * 3 + 1];
        char charen[16 + 1];
        int hof = 0;

        int cof = 0;
        int i;

        for (i = of; i < (int) of + 16 && i < (int) len; i++) {
            hof += sprintf(hexen + hof, "%02x ", p[i] & 0xff);
            cof += sprintf(charen + cof, "%c", isprint((int) p[i]) ? p[i] : '.');
        }
        fprintf(fp, "%08zx: %-48s %-16s\n", of, hexen, charen);
    }
}

static void msg_consume(rd_kafka_message_t *rkmessage, void *opaque)
{
    uint64_t time_stamp;

    struct timeval tnow;
    gettimeofday( &tnow, NULL );
    

    time_stamp = tnow.tv_sec*1000000 + tnow.tv_usec;
    
    if (rkmessage->err) {
        if (rkmessage->err == RD_KAFKA_RESP_ERR__PARTITION_EOF) {
            KAFKA_DEBUG("%% Consumer reached end of %s [%"PRId32"] "
                        "message queue at offset %"PRId64"\n",
                        rd_kafka_topic_name(rkmessage->rkt),
                        rkmessage->partition, rkmessage->offset);

            return;
        }

        KAFKA_DEBUG( "%% Consume error for topic \"%s\" [%"PRId32"] "
                     "offset %"PRId64": %s\n",
                     rd_kafka_topic_name(rkmessage->rkt),
                     rkmessage->partition,
                     rkmessage->offset,
                     rd_kafka_message_errstr(rkmessage));

        return;
    }

    if (!quiet) {
        rd_kafka_timestamp_type_t tstype;
        int64_t timestamp;
        fprintf(stdout, "%% Message (offset %"PRId64", %zd bytes):\n",
                rkmessage->offset, rkmessage->len);

        timestamp = rd_kafka_message_timestamp(rkmessage, &tstype);
        if (tstype != RD_KAFKA_TIMESTAMP_NOT_AVAILABLE) {
            const char *tsname = "?";
            if (tstype == RD_KAFKA_TIMESTAMP_CREATE_TIME)
                tsname = "create time";
            else if (tstype == RD_KAFKA_TIMESTAMP_LOG_APPEND_TIME)
                tsname = "log append time";

            fprintf(stdout, "%% Message timestamp: %s %"PRId64
                    " (%ds ago)\n",
                    tsname, timestamp,
                    !timestamp ? 0 :
                    (int)time(NULL) - (int)(timestamp/1000));
        }
    }

    if (rkmessage->key_len) {
        if (output == OUTPUT_HEXDUMP)
            hexdump(stdout, "Message Key",
                    rkmessage->key, rkmessage->key_len);
        else
            printf("Key: %.*s\n",
                   (int)rkmessage->key_len, (char *)rkmessage->key);
    }

    if (output == OUTPUT_HEXDUMP)
        hexdump(stdout, "Message Payload",
                rkmessage->payload, rkmessage->len);
    else
        printf("%.*s\n",
               (int)rkmessage->len, (char *)rkmessage->payload);

    KAFKA_DEBUG("Recieve timestamp=%lu\n", time_stamp);
}

/**
 * Kafka logger callback (optional)
 */
static void kafka_logger (const rd_kafka_t *rk, int level,
                          const char *fac, const char *buf) 
{
    struct timeval tv;
    gettimeofday(&tv, NULL);
    KAFKA_DEBUG("%u.%03u RDKAFKA-%i-%s: %s: %s\n",
                (int)tv.tv_sec, (int)(tv.tv_usec / 1000),
                level, fac, rk ? rd_kafka_name(rk) : NULL, buf);
}

/**
 * Message delivery report callback.
 * Called once for each message.
 * See rdkafka.h for more information.
 */
static void msg_delivered (rd_kafka_t *rk,
                           void *payload, size_t len,
                           int error_code,
                           void *opaque, void *msg_opaque) {

    if (error_code) {
        KAFKA_DEBUG("%% Message delivery failed: %s\n",
                    rd_kafka_err2str(error_code));
    } else if (!quiet) {
        KAFKA_DEBUG("%% Message delivered (%zd bytes): %.*s\n", len,
                    (int)len, (const char *)payload);
    }
}

/**
 * Message delivery report callback using the richer rd_kafka_message_t object.
 */
static void msg_delivered2 (rd_kafka_t *rk,
                            const rd_kafka_message_t *rkmessage, void *opaque) {
    KAFKA_DEBUG("del: %s: offset %"PRId64"\n",
                rd_kafka_err2str(rkmessage->err), rkmessage->offset);
    if (rkmessage->err) {
        KAFKA_DEBUG("%% Message delivery failed: %s\n",
                    rd_kafka_message_errstr(rkmessage));
    } else if (!quiet) {
        KAFKA_DEBUG("%% Message delivered (%zd bytes, offset %"PRId64", "
                    "partition %"PRId32"): %.*s\n",
                    rkmessage->len, rkmessage->offset,
                    rkmessage->partition,
                    (int)rkmessage->len, (const char *)rkmessage->payload);
    }
}

int ccmts_infra_kafka_create_producer(char *brokers, char *topic, rd_kafka_t **rk, 
                                rd_kafka_topic_t **rkt)
{
    int report_offsets = 0;
    char errstr[512];
    char tmp[16];
    static rd_kafka_conf_t *kafka_conf;
    rd_kafka_conf_res_t rc;

    /* Kafka configuration */
    kafka_conf = rd_kafka_conf_new();

    /* Set logger */
    rd_kafka_conf_set_log_cb(kafka_conf, kafka_logger);

    /* Quick termination */
    snprintf(tmp, sizeof(tmp), "%i", SIGIO);
    rd_kafka_conf_set(kafka_conf, 
                      "internal.termination.signal", tmp, NULL, 0);

    rd_kafka_topic_conf_t *topic_conf;
    /* Topic configuration */
    topic_conf = rd_kafka_topic_conf_new();

    /* Set up a message delivery report callback.
     * It will be called once for each message, either on successful
     * delivery to broker, or upon failure to deliver to broker. */

    /* If offset reporting (-o report) is enabled, use the
     * richer dr_msg_cb instead. */
    if (report_offsets) {
        rd_kafka_topic_conf_set(topic_conf,
                                "produce.offset.report",
                                "true", errstr, sizeof(errstr));
        rd_kafka_conf_set_dr_msg_cb(kafka_conf, msg_delivered2);
    } else
        rd_kafka_conf_set_dr_cb(kafka_conf, msg_delivered);

    /* Configure queue.buffering.max.ms to 0 to minimize the latency */
    rc = rd_kafka_conf_set(kafka_conf,
                            "queue.buffering.max.ms",
                            "0", errstr, sizeof(errstr));
    switch (rc) {
    case RD_KAFKA_CONF_UNKNOWN:
        KAFKA_DEBUG("%% Failed to set kafka_conf due to RD_KAFKA_CONF_UNKNOWN\n");
        break;
    case RD_KAFKA_CONF_INVALID:
        KAFKA_DEBUG("%% Failed to set kafka_conf due to RD_KAFKA_CONF_INVALID\n");
        break;
    case RD_KAFKA_CONF_OK:
        KAFKA_DEBUG("%% Succeeded to set kafka_conf \n");
        break;
    default:
        KAFKA_DEBUG("%% failed to set kafka_conf, rc=%d \n", rc);
        break;
    }

    /* Create Kafka handle */
    if (!(*rk = rd_kafka_new(RD_KAFKA_PRODUCER, kafka_conf,
                             errstr, sizeof(errstr)))) {
        KAFKA_DEBUG("%% Failed to create new producer: %s\n",
                    errstr);
        return FALSE;
    }

    rd_kafka_set_log_level(*rk, LOG_DEBUG);

    /* Add brokers */
    if (rd_kafka_brokers_add(*rk, brokers) == 0) {
        KAFKA_DEBUG("%% No valid brokers specified\n");
        return FALSE;
    }

    /* Create topic */
    *rkt = rd_kafka_topic_new(*rk, topic, topic_conf);
    KAFKA_DEBUG("%s, topic %s created \n", __FUNCTION__, topic);
    topic_conf = NULL; /* Now owned by topic */
    return TRUE;
}

/*
 * Consumer
 */
int ccmts_infra_kafka_create_consumer(char *brokers, char *topic, rd_kafka_t **rk,
                                rd_kafka_topic_t **rkt)
{
    char errstr[512];
    rd_kafka_topic_conf_t *topic_conf;
    char tmp[16];
    static rd_kafka_conf_t *kafka_conf;

    /* Kafka configuration */
    kafka_conf = rd_kafka_conf_new();

    /* Set logger */
    rd_kafka_conf_set_log_cb(kafka_conf, kafka_logger);

    /* Quick termination */
    snprintf(tmp, sizeof(tmp), "%i", SIGIO);
    rd_kafka_conf_set(kafka_conf, 
                      "internal.termination.signal", tmp, NULL, 0);


    /* Topic configuration */
    topic_conf = rd_kafka_topic_conf_new();

    /* Create Kafka handle */
    if (!(*rk = rd_kafka_new(RD_KAFKA_CONSUMER, kafka_conf,
                             errstr, sizeof(errstr)))) {
        KAFKA_DEBUG("%% Failed to create new consumer: %s\n",
                    errstr);
        return FALSE;
    }

    rd_kafka_set_log_level(*rk, LOG_DEBUG);

    /* Add brokers */
    if (rd_kafka_brokers_add(*rk, brokers) == 0) {
        KAFKA_DEBUG("%% No valid brokers specified\n");
        return FALSE;
    }

    /* Create topic */
    *rkt = rd_kafka_topic_new(*rk, topic, topic_conf);
    topic_conf = NULL; /* Now owned by topic */
    return TRUE;
}

int ccmts_infra_kafka_send_msg(rd_kafka_t *rk, 
                         rd_kafka_topic_t *rkt, 
                         int partition, char *buf, int len)
{

    /* Send/Produce message. */
    if (rd_kafka_produce(rkt, partition,
                         RD_KAFKA_MSG_F_COPY,
                         /* Payload and length */
                         buf, len,
                         /* Optional key and its length */
                         NULL, 0,
                         /* Message opaque, provided in
                          * delivery report callback as
                          * msg_opaque. */
                         NULL) == -1) {
        KAFKA_DEBUG("%% Failed to produce to topic %s "
                    "partition %i: %s\n",
                    rd_kafka_topic_name(rkt), partition,
                    rd_kafka_err2str(rd_kafka_last_error()));
        /* Poll to handle delivery reports */
        rd_kafka_poll(rk, 0);
        return FALSE;
    }
    /* Poll to handle delivery reports */
    rd_kafka_poll(rk, 0);
    return TRUE;
}

int ccmts_infra_kafka_consume_msg(rd_kafka_t *rk, rd_kafka_topic_t *rkt)
{
    rd_kafka_message_t *rkmessage;
    int partition = 0;

    /* Start consuming */
    if (rd_kafka_consume_start(rkt, partition, RD_KAFKA_OFFSET_END) == -1) {
        rd_kafka_resp_err_t err = rd_kafka_last_error();
        KAFKA_DEBUG("%% Failed to start consuming: %s\n", rd_kafka_err2str(err));
        if (err == RD_KAFKA_RESP_ERR__INVALID_ARG)
            KAFKA_DEBUG("%% Broker based offset storage "
                        "requires a group.id, "
                        "add: -X group.id=yourGroup\n");
        exit(1);
    }


    /* Consume single message.
     * See rdkafka_performance.c for high speed
     * consuming of messages. */
    KAFKA_DEBUG("%s:%d\n", __FUNCTION__, __LINE__);
    while (TRUE) {

        rkmessage = rd_kafka_consume(rkt, partition, 1000);
        if (!rkmessage) /* timeout */
        {
            continue;
        }

        msg_consume(rkmessage, NULL);
        rd_kafka_commit_message(rk, rkmessage, 0);

        /* Return message to rdkafka */
        rd_kafka_message_destroy(rkmessage);
    }

    /* Stop consuming */
    rd_kafka_consume_stop(rkt, partition);

    while (rd_kafka_outq_len(rk) > 0)
        rd_kafka_poll(rk, 10);
    return TRUE;

}

int ccmts_infra_kafka_consume_callback_queue(rd_kafka_t *rk, rd_kafka_topic_t *rkt,
                                             rd_kafka_queue_t *rkqu)
{
    /* Consume messages.
     * A message may either be a real message, or
     * an error signaling (if rkmessage->err is set).
     */
    int r;

    while (TRUE) {
        /* Poll for errors, etc. */
        rd_kafka_poll(rk, 0);

        r = rd_kafka_consume_callback_queue(rkqu, 1000,
                                            msg_consume,
                                            NULL);
        if (r == -1) {
            KAFKA_DEBUG("%% Error: %s\n",
                        rd_kafka_err2str(rd_kafka_errno2err(errno)));
        }
    }

    return TRUE;

}

int ccmts_infra_kafka_consume_single_msg(rd_kafka_t *rk, 
                            rd_kafka_topic_t *rkt )
{
    rd_kafka_message_t *rkmessage;
    int partition = RD_KAFKA_PARTITION_UA;

    /* Start consuming */
    if (rd_kafka_consume_start(rkt, partition, RD_KAFKA_OFFSET_END) == -1) {
        rd_kafka_resp_err_t err = rd_kafka_last_error();
        KAFKA_DEBUG("%% Failed to start consuming: %s\n", rd_kafka_err2str(err));
        if (err == RD_KAFKA_RESP_ERR__INVALID_ARG)
            KAFKA_DEBUG("%% Broker based offset storage "
                        "requires a group.id, "
                        "add: -X group.id=yourGroup\n");
        exit(1);
    }

    /* Consume single message.
     * See rdkafka_performance.c for high speed
     * consuming of messages. */
    while (TRUE) {
        rkmessage = rd_kafka_consume(rkt, partition, RD_KAFKA_OFFSET_END);
        if (!rkmessage) /* timeout */
            continue;
    }
    msg_consume(rkmessage, NULL);

    /* Return message to rdkafka */
    rd_kafka_message_destroy(rkmessage);

    /* Stop consuming */
    rd_kafka_consume_stop(rkt, partition);

    while (rd_kafka_outq_len(rk) > 0)
        rd_kafka_poll(rk, 10);
    return TRUE;

}

