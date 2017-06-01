#include <stdio.h>
#include <stdlib.h>
#include "microhttpd.h"
#include <errno.h>
#include <string.h>
#include <sys/time.h>
#include <sys/resource.h>
#include <pthread.h>
#include "ccmts_infra_kafka.h"
#include "ev.h"
#include "ccmts_types.h"
#include "event.h"

#define port 80

char *topic_test = "ccmts_sample_test";
rd_kafka_queue_t *rkqu = NULL;

static char *broker_d = "kafka-0.kafka.default.svc.cluster.local:9092,"
"kafka-1.kafka.default.svc.cluster.local:9092,"
"kafka-2.kafka.default.svc.cluster.local:9092";

struct ev_loop *main_loop = NULL;

rd_kafka_t *rk_producer_test;
rd_kafka_t *rk_consumer_test;
rd_kafka_topic_t *rkt_producer_test;
rd_kafka_topic_t *rkt_consumer_test;

extern int ccmts_cassandra_test();

#define CCMTS_DEBUG(format, ...) \
{\
    fprintf(stdout, format, ##__VA_ARGS__);\
    fflush(stdout);\
}


#define PAGE "<html><head><title>libmicrohttpd demo</title>"\
             "</head><body>libmicrohttpd demo</body></html>"


void ccmts_kafka_test()
{
    char *broker = getenv(KAFKA_BROKER);
    char *test = "This is a one time test message for kafka";

    if (!broker) {
        broker = broker_d;
    }
    CCMTS_DEBUG("kafka broker is %s \r\n", broker);

    /* Create a kafka producer */
    ccmts_infra_kafka_create_producer(broker, topic_test, 
                                      &rk_producer_test,
                                      &rkt_producer_test);
    /* Send a message to test topic, partition 0 */
    ccmts_infra_kafka_send_msg(rk_producer_test,
                               rkt_producer_test,
                               0,
                               test, strlen(test));

    /* Send a message to test topic, partition 1 */
    ccmts_infra_kafka_send_msg(rk_producer_test,
                               rkt_producer_test,
                               1,
                               test, strlen(test));

}
