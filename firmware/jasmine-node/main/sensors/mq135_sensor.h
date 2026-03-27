#pragma once

#include "esp_err.h"
#include "mqtt_client.h"
#include <esp_adc/adc_oneshot.h>

typedef struct {
    adc_oneshot_unit_handle_t adc_handle;
} mq135_ctx_t;

esp_err_t mq135_sensor_init(void *ctx);
esp_err_t mq135_sensor_read(void *ctx, esp_mqtt_client_handle_t client);
