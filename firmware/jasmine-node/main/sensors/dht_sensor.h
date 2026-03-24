#pragma once

#include "esp_err.h"
#include "mqtt_client.h"

typedef struct {
    int dummy;
} dht_ctx_t;

esp_err_t dht_sensor_init(void *ctx);
esp_err_t dht_sensor_read(void *ctx, esp_mqtt_client_handle_t client);
