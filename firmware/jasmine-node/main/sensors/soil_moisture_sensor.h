#pragma once

#include "esp_err.h"
#include "mqtt_client.h"
#include <esp_adc/adc_oneshot.h>

typedef struct {
    adc_oneshot_unit_handle_t adc_handle;
} soil_moisture_ctx_t;

esp_err_t soil_moisture_sensor_init(void *ctx);
esp_err_t soil_moisture_sensor_read(void *ctx, esp_mqtt_client_handle_t client);
