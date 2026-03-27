#include "mq135_sensor.h"
#include "mqtt.h"
#include "mq135.h"

#include "esp_log.h"

#define READING_BUF_SIZE 32

static const char *TAG = "mq135";

static mq135_t s_mq135_ctx;

esp_err_t mq135_sensor_init(void *ctx) {
  mq135_ctx_t *mq = (mq135_ctx_t *)ctx;

  mq135_config_t config = {
      .sample_count = CONFIG_MQ135_SAMPLE_COUNT,
      .sample_delay_ms = CONFIG_MQ135_SAMPLE_DELAY_MS,
      .valid_min = CONFIG_MQ135_VALID_MIN,
      .valid_max = CONFIG_MQ135_VALID_MAX,
      .preheat_ms = CONFIG_MQ135_PREHEAT_MS,
      .rl_kohm = CONFIG_MQ135_RL_KOHM_X100 / 100.0f,
      .r0 = CONFIG_MQ135_R0_X100 / 100.0f,
      .voltage_ref = CONFIG_MQ135_VOLTAGE_REF_MV / 1000.0f,
      .co2_a = 110.47f,
      .co2_b = -2.862f,
  };

  return mq135_setup(&s_mq135_ctx, mq->adc_handle,
                     CONFIG_MQ135_ADC1_CH, &config);
}

esp_err_t mq135_sensor_read(void *ctx, esp_mqtt_client_handle_t client) {
  float ppm;
  uint32_t raw;
  esp_err_t err = mq135_read_ppm(&s_mq135_ctx, &ppm, &raw);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "mq135_read_ppm failed: %s", esp_err_to_name(err));
    return err;
  }

  ESP_LOGI(TAG, "CO2: %.2f ppm | Raw: %lu", ppm, (unsigned long)raw);

  char buf[READING_BUF_SIZE];
  int msg_id;

  snprintf(buf, sizeof(buf), "%.2f", ppm);
  msg_id = mqtt_send_reading(client, "co2", buf);
  if (msg_id < 0) {
    return ESP_FAIL;
  }

  snprintf(buf, sizeof(buf), "%lu", (unsigned long)raw);
  msg_id = mqtt_send_reading(client, "co2-raw", buf);
  return msg_id >= 0 ? ESP_OK : ESP_FAIL;
}
