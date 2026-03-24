#include "dht_sensor.h"
#include "mqtt.h"

#include "dht.h"
#include "driver/gpio.h"
#include "esp_log.h"

#define DHT_SENSOR_TYPE DHT_TYPE_AM2301
#define READING_BUF_SIZE 32

_Static_assert(CONFIG_DHT_DATA_GPIO < GPIO_NUM_MAX,
               "CONFIG_DHT_DATA_GPIO exceeds valid GPIO range for this SoC");

static const char *TAG = "dht";

esp_err_t dht_sensor_init(void *ctx) {
  return ESP_OK;
}

esp_err_t dht_sensor_read(void *ctx, esp_mqtt_client_handle_t client) {
  float temperature, humidity;

  esp_err_t err = dht_read_float_data(DHT_SENSOR_TYPE,
                                      (gpio_num_t)CONFIG_DHT_DATA_GPIO,
                                      &humidity, &temperature);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "Failed to read sensor: %s", esp_err_to_name(err));
    return err;
  }

  ESP_LOGI(TAG, "Temperature: %.1f C | Humidity: %.1f%%", temperature,
           humidity);

  char buf[READING_BUF_SIZE];
  int msg_id;

  snprintf(buf, sizeof(buf), "%.2f", humidity);
  msg_id = mqtt_send_reading(client, "humidity", buf);
  if (msg_id < 0) {
    return ESP_FAIL;
  }

  snprintf(buf, sizeof(buf), "%.2f", temperature);
  msg_id = mqtt_send_reading(client, "temperature", buf);
  return msg_id >= 0 ? ESP_OK : ESP_FAIL;
}
