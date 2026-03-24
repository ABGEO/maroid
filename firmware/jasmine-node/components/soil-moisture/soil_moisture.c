#include "soil_moisture.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

static const adc_bitwidth_t bitwidth = ADC_BITWIDTH_DEFAULT;
static const adc_atten_t attenuation = ADC_ATTEN_DB_12;

esp_err_t soil_moisture_setup(soil_moisture_t *ctx,
                              adc_oneshot_unit_handle_t adc_handle,
                              adc_channel_t channel,
                              const soil_moisture_config_t *config) {
  if (ctx == NULL || config == NULL) {
    return ESP_ERR_INVALID_ARG;
  }

  if (config->sample_count == 0 ||
      config->sample_count > SOIL_MOISTURE_MAX_SAMPLE_COUNT) {
    return ESP_ERR_INVALID_ARG;
  }

  if (config->cal_air_value == config->cal_water_value) {
    return ESP_ERR_INVALID_ARG;
  }

  ctx->adc_handle = adc_handle;
  ctx->channel = channel;
  ctx->config = *config;

  adc_oneshot_chan_cfg_t chan_cfg = {
      .bitwidth = bitwidth,
      .atten = attenuation,
  };

  return adc_oneshot_config_channel(ctx->adc_handle, ctx->channel, &chan_cfg);
}

esp_err_t soil_moisture_read(const soil_moisture_t *ctx, uint32_t *out_raw) {
  if (ctx == NULL || out_raw == NULL) {
    return ESP_ERR_INVALID_ARG;
  }

  int adc_raw;
  uint32_t accumulator = 0;
  uint32_t valid_count = 0;

  for (uint32_t i = 0; i < ctx->config.sample_count; i++) {
    if (adc_oneshot_read(ctx->adc_handle, ctx->channel, &adc_raw) == ESP_OK) {
      uint32_t raw = (uint32_t)adc_raw;
      if (raw >= ctx->config.valid_min && raw <= ctx->config.valid_max) {
        accumulator += raw;
        valid_count++;
      }
    }

    if (ctx->config.sample_delay_ms > 0 &&
        i < ctx->config.sample_count - 1) {
      vTaskDelay(pdMS_TO_TICKS(ctx->config.sample_delay_ms));
    }
  }

  if (valid_count == 0) {
    return ESP_ERR_INVALID_STATE;
  }

  *out_raw = accumulator / valid_count;
  return ESP_OK;
}

float soil_moisture_normalize(const soil_moisture_t *ctx, uint32_t raw_value) {
  if (ctx == NULL) {
    return 0.0f;
  }

  float air = (float)ctx->config.cal_air_value;
  float water = (float)ctx->config.cal_water_value;
  float raw = (float)raw_value;

  float percentage = 100.0f - ((raw - water) / (air - water)) * 100.0f;

  if (percentage < 0.0f) {
    percentage = 0.0f;
  } else if (percentage > 100.0f) {
    percentage = 100.0f;
  }

  return percentage;
}
