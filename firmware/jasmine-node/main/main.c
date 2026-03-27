#include "esp_err.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_sleep.h"
#include "esp_wifi.h"

#include "mqtt.h"
#include "nvs.h"
#include "sensor_registry.h"
#include "sensor_task.h"
#include "sntp.h"
#include "wifi.h"

#ifdef CONFIG_SENSOR_BH1750_ENABLED
#include "i2c.h"
#endif

#if defined(CONFIG_SENSOR_SOIL_MOISTURE_ENABLED) || defined(CONFIG_SENSOR_MQ135_ENABLED)
#include "adc.h"
#endif

static const char *TAG = "main";

void app_main(void) {
  ESP_ERROR_CHECK(nvs_setup());
  ESP_ERROR_CHECK(esp_netif_init());
  ESP_ERROR_CHECK(esp_event_loop_create_default());
  ESP_ERROR_CHECK(wifi_setup());
  ESP_ERROR_CHECK(sntp_setup());

#ifdef CONFIG_SENSOR_BH1750_ENABLED
  i2c_master_bus_handle_t i2c_bus;
  ESP_ERROR_CHECK(i2c_master_bus_init(&i2c_bus));
  ESP_LOGI(TAG, "I2C initialized");
  sensor_registry_bh1750_ctx()->i2c_bus = i2c_bus;
#endif

#if defined(CONFIG_SENSOR_SOIL_MOISTURE_ENABLED) || defined(CONFIG_SENSOR_MQ135_ENABLED)
  adc_oneshot_unit_handle_t adc1_handle;
  ESP_ERROR_CHECK(adc_oneshot_unit_init(&adc1_handle));
  ESP_LOGI(TAG, "ADC initialized");

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
  sensor_registry_soil_moisture_ctx()->adc_handle = adc1_handle;
#endif

#ifdef CONFIG_SENSOR_MQ135_ENABLED
  sensor_registry_mq135_ctx()->adc_handle = adc1_handle;
#endif
#endif

  esp_mqtt_client_handle_t mqtt_client;
  esp_err_t err = mqtt_start(&mqtt_client);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "MQTT failed, skipping to deep sleep");
    esp_wifi_stop();
    esp_wifi_deinit();

    return;
  }

  sensors_run(mqtt_client);
  mqtt_wait_published();
  mqtt_stop(mqtt_client);
  esp_wifi_stop();
  esp_wifi_deinit();

  ESP_LOGI(TAG, "Entering deep sleep for %d s", CONFIG_SLEEP_DURATION_SEC);
  esp_sleep_enable_timer_wakeup((uint64_t)CONFIG_SLEEP_DURATION_SEC *
                                1000000ULL);
  esp_deep_sleep_start();
}
