#include "soil_moisture.h"

static const adc_bitwidth_t bitwidth = ADC_BITWIDTH_DEFAULT;
static const adc_atten_t attenuation = ADC_ATTEN_DB_12;

esp_err_t soil_moisture_setup(soil_moisture_t *ctx,
                              adc_oneshot_unit_handle_t adc_handle,
                              adc_channel_t channel) {
  ctx->adc_handle = adc_handle;
  ctx->channel = channel;

  adc_oneshot_chan_cfg_t config = {
      .bitwidth = bitwidth,
      .atten = attenuation,
  };

  return adc_oneshot_config_channel(ctx->adc_handle, ctx->channel, &config);
}

uint32_t soil_moisture_read(const soil_moisture_t *ctx) {
  int adc_raw;
  int dividers = 0;
  uint32_t adc_reading = 0;

  for (int i = 0; i < NO_OF_SAMPLES; i++) {
    if (adc_oneshot_read(ctx->adc_handle, ctx->channel, &adc_raw) == ESP_OK) {
      adc_reading += adc_raw;
      dividers++;
    }
  }

  return dividers != 0 ? adc_reading / dividers : adc_reading;
}

float soil_moisture_normalize(uint32_t min_value, uint32_t max_value,
                              uint32_t value) {
  return 100.0f -
         ((float)(value - min_value) / (float)(max_value - min_value)) * 100.0f;
}
