#include "mq135.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include <math.h>

static const adc_bitwidth_t bitwidth = ADC_BITWIDTH_DEFAULT;
static const adc_atten_t attenuation = ADC_ATTEN_DB_12;

esp_err_t mq135_setup(mq135_t *ctx,
                      adc_oneshot_unit_handle_t adc_handle,
                      adc_channel_t channel,
                      const mq135_config_t *config) {
  if (ctx == NULL || config == NULL) {
    return ESP_ERR_INVALID_ARG;
  }

  if (config->sample_count == 0 ||
      config->sample_count > MQ135_MAX_SAMPLE_COUNT) {
    return ESP_ERR_INVALID_ARG;
  }

  if (config->r0 <= 0.0f || config->rl_kohm <= 0.0f) {
    return ESP_ERR_INVALID_ARG;
  }

  if (config->co2_a <= 0.0f) {
    return ESP_ERR_INVALID_ARG;
  }

  ctx->adc_handle = adc_handle;
  ctx->channel = channel;
  ctx->config = *config;
  ctx->preheated = false;

  adc_oneshot_chan_cfg_t chan_cfg = {
      .bitwidth = bitwidth,
      .atten = attenuation,
  };

  return adc_oneshot_config_channel(ctx->adc_handle, ctx->channel, &chan_cfg);
}

esp_err_t mq135_read_raw(const mq135_t *ctx, uint32_t *out_raw) {
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

esp_err_t mq135_read_ppm(mq135_t *ctx, float *out_ppm, uint32_t *out_raw) {
  if (ctx == NULL || out_ppm == NULL) {
    return ESP_ERR_INVALID_ARG;
  }

  if (!ctx->preheated && ctx->config.preheat_ms > 0) {
    vTaskDelay(pdMS_TO_TICKS(ctx->config.preheat_ms));
    ctx->preheated = true;
  }

  uint32_t raw;
  esp_err_t err = mq135_read_raw(ctx, &raw);
  if (err != ESP_OK) {
    return err;
  }

  if (out_raw != NULL) {
    *out_raw = raw;
  }

  float vout = ((float)raw / 4095.0f) * ctx->config.voltage_ref;

  if (vout <= 0.0f) {
    return ESP_ERR_INVALID_STATE;
  }

  float rs = ctx->config.rl_kohm * (ctx->config.voltage_ref - vout) / vout;

  if (rs <= 0.0f) {
    return ESP_ERR_INVALID_STATE;
  }

  float ratio = rs / ctx->config.r0;
  float ppm = ctx->config.co2_a * powf(ratio, ctx->config.co2_b);

  if (ppm < 0.0f) {
    ppm = 0.0f;
  }

  *out_ppm = ppm;
  return ESP_OK;
}
