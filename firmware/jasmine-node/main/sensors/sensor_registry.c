#include "sensor_registry.h"

#ifdef CONFIG_SENSOR_BH1750_ENABLED
static bh1750_ctx_t s_bh1750_ctx;
#endif

#ifdef CONFIG_SENSOR_DHT_ENABLED
static dht_ctx_t s_dht_ctx;
#endif

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
static soil_moisture_ctx_t s_soil_moisture_ctx;
#endif

#ifdef CONFIG_SENSOR_MQ135_ENABLED
static mq135_ctx_t s_mq135_ctx;
#endif

static const sensor_descriptor_t s_sensors[] = {
#ifdef CONFIG_SENSOR_BH1750_ENABLED
    {
        .name = "bh1750",
        .init = bh1750_sensor_init,
        .read = bh1750_sensor_read,
        .cleanup = bh1750_sensor_cleanup,
        .ctx = &s_bh1750_ctx,
        .stack_size = 4096,
    },
#endif
#ifdef CONFIG_SENSOR_DHT_ENABLED
    {
        .name = "dht",
        .init = dht_sensor_init,
        .read = dht_sensor_read,
        .cleanup = NULL,
        .ctx = &s_dht_ctx,
        .stack_size = 4096,
    },
#endif
#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
    {
        .name = "soil_moisture",
        .init = soil_moisture_sensor_init,
        .read = soil_moisture_sensor_read,
        .cleanup = NULL,
        .ctx = &s_soil_moisture_ctx,
        .stack_size = 4096,
    },
#endif
#ifdef CONFIG_SENSOR_MQ135_ENABLED
    {
        .name = "mq135",
        .init = mq135_sensor_init,
        .read = mq135_sensor_read,
        .cleanup = NULL,
        .ctx = &s_mq135_ctx,
        .stack_size = 8192,
    },
#endif
};

#define SENSOR_COUNT (sizeof(s_sensors) / sizeof(s_sensors[0]))

_Static_assert(SENSOR_COUNT <= 12,
               "Maximum 12 sensors supported (FreeRTOS EventBits_t limit)");

size_t sensor_count(void) { return SENSOR_COUNT; }

const sensor_descriptor_t *sensor_list(void) { return s_sensors; }

#ifdef CONFIG_SENSOR_BH1750_ENABLED
bh1750_ctx_t *sensor_registry_bh1750_ctx(void) { return &s_bh1750_ctx; }
#endif

#ifdef CONFIG_SENSOR_DHT_ENABLED
dht_ctx_t *sensor_registry_dht_ctx(void) { return &s_dht_ctx; }
#endif

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
soil_moisture_ctx_t *sensor_registry_soil_moisture_ctx(void) {
  return &s_soil_moisture_ctx;
}
#endif

#ifdef CONFIG_SENSOR_MQ135_ENABLED
mq135_ctx_t *sensor_registry_mq135_ctx(void) { return &s_mq135_ctx; }
#endif
