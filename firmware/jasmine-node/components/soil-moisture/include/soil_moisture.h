#pragma once

#include "esp_err.h"
#include <esp_adc/adc_oneshot.h>
#include <hal/adc_types.h>

#define SOIL_MOISTURE_MAX_SAMPLE_COUNT 1024

typedef struct {
    uint32_t sample_count;
    uint32_t sample_delay_ms;
    uint32_t valid_min;
    uint32_t valid_max;
    uint32_t cal_air_value;
    uint32_t cal_water_value;
} soil_moisture_config_t;

typedef struct {
    adc_oneshot_unit_handle_t adc_handle;
    adc_channel_t channel;
    soil_moisture_config_t config;
} soil_moisture_t;

esp_err_t soil_moisture_setup(soil_moisture_t *ctx,
                              adc_oneshot_unit_handle_t adc_handle,
                              adc_channel_t channel,
                              const soil_moisture_config_t *config);
esp_err_t soil_moisture_read(const soil_moisture_t *ctx, uint32_t *out_raw);
float soil_moisture_normalize(const soil_moisture_t *ctx, uint32_t raw_value);
