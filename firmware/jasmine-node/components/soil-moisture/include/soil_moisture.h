#pragma once

#include <esp_adc/adc_oneshot.h>
#include <hal/adc_types.h>

#define NO_OF_SAMPLES 64

typedef struct {
  adc_oneshot_unit_handle_t adc_handle;
  adc_channel_t channel;
} soil_moisture_t;

esp_err_t soil_moisture_setup(soil_moisture_t *ctx,
                              adc_oneshot_unit_handle_t adc_handle,
                              adc_channel_t channel);
uint32_t soil_moisture_read(const soil_moisture_t *ctx);
float soil_moisture_normalize(uint32_t min_value, uint32_t max_value,
                              uint32_t value);
