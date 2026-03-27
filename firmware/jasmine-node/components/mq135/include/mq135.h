#pragma once

#include "esp_err.h"
#include <esp_adc/adc_oneshot.h>
#include <hal/adc_types.h>
#include <stdbool.h>

#define MQ135_MAX_SAMPLE_COUNT 1024

typedef struct {
    uint32_t sample_count;
    uint32_t sample_delay_ms;
    uint32_t valid_min;
    uint32_t valid_max;
    uint32_t preheat_ms;
    float rl_kohm;
    float r0;
    float voltage_ref;
    float co2_a;
    float co2_b;
} mq135_config_t;

typedef struct {
    adc_oneshot_unit_handle_t adc_handle;
    adc_channel_t channel;
    mq135_config_t config;
    bool preheated;
} mq135_t;

esp_err_t mq135_setup(mq135_t *ctx,
                      adc_oneshot_unit_handle_t adc_handle,
                      adc_channel_t channel,
                      const mq135_config_t *config);
esp_err_t mq135_read_raw(const mq135_t *ctx, uint32_t *out_raw);
esp_err_t mq135_read_ppm(mq135_t *ctx, float *out_ppm, uint32_t *out_raw);
