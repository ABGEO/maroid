#pragma once

#include "esp_err.h"
#include "mqtt_client.h"
#include <stddef.h>

typedef esp_err_t (*sensor_init_fn)(void *ctx);
typedef esp_err_t (*sensor_read_fn)(void *ctx, esp_mqtt_client_handle_t client);
typedef void (*sensor_cleanup_fn)(void *ctx);

typedef struct {
    const char *name;
    sensor_init_fn init;
    sensor_read_fn read;
    sensor_cleanup_fn cleanup;
    void *ctx;
    uint32_t stack_size;
} sensor_descriptor_t;

size_t sensor_count(void);
const sensor_descriptor_t *sensor_list(void);

#ifdef CONFIG_SENSOR_BH1750_ENABLED
#include "bh1750_sensor.h"
bh1750_ctx_t *sensor_registry_bh1750_ctx(void);
#endif

#ifdef CONFIG_SENSOR_DHT_ENABLED
#include "dht_sensor.h"
dht_ctx_t *sensor_registry_dht_ctx(void);
#endif

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
#include "soil_moisture_sensor.h"
soil_moisture_ctx_t *sensor_registry_soil_moisture_ctx(void);
#endif
