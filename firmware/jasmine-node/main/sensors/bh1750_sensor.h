#pragma once

#include "driver/i2c_master.h"
#include "esp_err.h"
#include "mqtt_client.h"

typedef struct {
    i2c_master_bus_handle_t i2c_bus;
} bh1750_ctx_t;

esp_err_t bh1750_sensor_init(void *ctx);
esp_err_t bh1750_sensor_read(void *ctx, esp_mqtt_client_handle_t client);
void bh1750_sensor_cleanup(void *ctx);
