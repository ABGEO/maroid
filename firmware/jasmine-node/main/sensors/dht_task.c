#include "dht_task.h"
#include "mqtt.h"

#include "dht.h"
#include "driver/gpio.h"
#include "esp_log.h"
#include "freertos/event_groups.h"
#include "freertos/task.h"

#define DHT_SENSOR_TYPE DHT_TYPE_AM2301
_Static_assert(CONFIG_DHT_DATA_GPIO < GPIO_NUM_MAX,
               "CONFIG_DHT_DATA_GPIO exceeds valid GPIO range for this SoC");

#define SENSOR_READING_BUF_SIZE 32

static const char *TAG = "dht_task";

void dht_task(void *arg) {
  dht_task_params_t *params = (dht_task_params_t *)arg;

  float temperature_reading, humidity_reading;
  char reading_buf[SENSOR_READING_BUF_SIZE];

  if (dht_read_float_data(DHT_SENSOR_TYPE, (gpio_num_t)CONFIG_DHT_DATA_GPIO,
                          &humidity_reading, &temperature_reading) == ESP_OK) {
    ESP_LOGI(TAG, "Temperature: %.1f C | Humidity: %.1f%%", temperature_reading,
             humidity_reading);

    snprintf(reading_buf, sizeof(reading_buf), "%f", humidity_reading);
    mqtt_send_reading(params->mqtt_client, "humidity", reading_buf);

    snprintf(reading_buf, sizeof(reading_buf), "%f", temperature_reading);
    mqtt_send_reading(params->mqtt_client, "temperature", reading_buf);

    xEventGroupSetBits(params->event_group, DHT_DONE_BIT);
  } else {
    ESP_LOGE(TAG, "Failed to read sensor");
    xEventGroupSetBits(params->event_group, DHT_DONE_BIT | DHT_ERR_BIT);
  }

  vTaskDelete(NULL);
}
