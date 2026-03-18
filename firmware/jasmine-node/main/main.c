#include "esp_err.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_sleep.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"
#include "freertos/task.h"

#include "mqtt.h"
#include "nvs_setup.h"
#include "sntp.h"
#include "wifi.h"

#ifdef CONFIG_SENSOR_BH1750_ENABLED
#include "bh1750_task.h"
#include "driver/i2c_master.h"
#endif

#ifdef CONFIG_SENSOR_DHT_ENABLED
#include "dht_task.h"
#endif

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
#include "adc.h"
#include "soil_moisture_task.h"
#endif

static const char *TAG = "main";

#ifdef CONFIG_SENSOR_BH1750_ENABLED
static esp_err_t i2c_master_init(i2c_master_bus_handle_t *bus_handle) {
  i2c_master_bus_config_t bus_config = {
      .i2c_port = I2C_NUM_0,
      .sda_io_num = CONFIG_I2C_MASTER_SDA_IO,
      .scl_io_num = CONFIG_I2C_MASTER_SCL_IO,
      .clk_source = I2C_CLK_SRC_DEFAULT,
      .glitch_ignore_cnt = 7,
      .flags.enable_internal_pullup = true,
  };

  return i2c_new_master_bus(&bus_config, bus_handle);
}
#endif

void app_main(void) {
  esp_sleep_wakeup_cause_t cause = esp_sleep_get_wakeup_cause();
  if (cause == ESP_SLEEP_WAKEUP_TIMER) {
    ESP_LOGI(TAG, "Wake from deep sleep (timer)");
  } else {
    ESP_LOGI(TAG, "Startup");
  }

#if CONFIG_LOG_MAXIMUM_LEVEL > CONFIG_LOG_DEFAULT_LEVEL
  esp_log_level_set("wifi", CONFIG_LOG_MAXIMUM_LEVEL);
#endif

  ESP_ERROR_CHECK(nvs_setup());
  ESP_ERROR_CHECK(esp_netif_init());
  ESP_ERROR_CHECK(esp_event_loop_create_default());
  ESP_ERROR_CHECK(wifi_setup());
  ESP_ERROR_CHECK(sntp_setup());

#ifdef CONFIG_SENSOR_BH1750_ENABLED
  i2c_master_bus_handle_t i2c_bus;
  ESP_ERROR_CHECK(i2c_master_init(&i2c_bus));
  ESP_LOGI(TAG, "I2C initialized");
#endif

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
  adc_oneshot_unit_handle_t adc1_handle;
  ESP_ERROR_CHECK(setup_adc_oneshot_unit(&adc1_handle));
  ESP_LOGI(TAG, "ADC initialized");
#endif

  EventGroupHandle_t sensor_event_group = xEventGroupCreate();
  if (sensor_event_group == NULL) {
    ESP_LOGE(TAG, "Failed to create sensor_event_group");
    abort();
  }

  esp_mqtt_client_handle_t mqtt_client;
  ESP_ERROR_CHECK(mqtt_start(&mqtt_client));

  EventBits_t all_done_bits = 0;

  /* Static storage ensures the pointer passed to xTaskCreate remains valid.
   * app_main also blocks on xEventGroupWaitBits until all tasks finish,
   * so stack-allocated would be safe too — static is belt-and-suspenders. */

#ifdef CONFIG_SENSOR_BH1750_ENABLED
  static bh1750_task_params_t bh1750_params;
  bh1750_params.mqtt_client = mqtt_client;
  bh1750_params.bus = i2c_bus;
  bh1750_params.event_group = sensor_event_group;

  if (xTaskCreate(bh1750_task, "bh1750", 4096, &bh1750_params, 5, NULL) !=
      pdPASS) {
    ESP_LOGE(TAG, "Failed to create bh1750_task");
    abort();
  }
  all_done_bits |= BH1750_DONE_BIT;
#endif

#ifdef CONFIG_SENSOR_DHT_ENABLED
  static dht_task_params_t dht_params;
  dht_params.mqtt_client = mqtt_client;
  dht_params.event_group = sensor_event_group;

  if (xTaskCreate(dht_task, "dht", 4096, &dht_params, 5, NULL) != pdPASS) {
    ESP_LOGE(TAG, "Failed to create dht_task");
    abort();
  }
  all_done_bits |= DHT_DONE_BIT;
#endif

#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
  static soil_moisture_task_params_t soil_moisture_params;
  soil_moisture_params.adc_handle = adc1_handle;
  soil_moisture_params.mqtt_client = mqtt_client;
  soil_moisture_params.event_group = sensor_event_group;

  if (xTaskCreate(soil_moisture_task, "soil_moisture", 4096,
                  &soil_moisture_params, 5, NULL) != pdPASS) {
    ESP_LOGE(TAG, "Failed to create soil_moisture");
    abort();
  }
  all_done_bits |= SOIL_MOISTURE_DONE_BIT;
#endif

  if (all_done_bits != 0) {
    EventBits_t done =
        xEventGroupWaitBits(sensor_event_group, all_done_bits,
                            pdFALSE, // don't clear — need to read error bits
                            pdTRUE,  // wait for ALL bits
                            portMAX_DELAY);

#ifdef CONFIG_SENSOR_BH1750_ENABLED
    if (done & BH1750_ERR_BIT)
      ESP_LOGW(TAG, "BH1750 sensor failed");
#endif
#ifdef CONFIG_SENSOR_DHT_ENABLED
    if (done & DHT_ERR_BIT)
      ESP_LOGW(TAG, "DHT sensor failed");
#endif
#ifdef CONFIG_SENSOR_SOIL_MOISTURE_ENABLED
    if (done & SOIL_MOISTURE_ERR_BIT)
      ESP_LOGW(TAG, "Soil moisture sensor failed");
#endif
  } else {
    ESP_LOGW(TAG, "No sensors enabled");
  }

  mqtt_wait_published();
  mqtt_stop(mqtt_client);

  ESP_LOGI(TAG, "Entering deep sleep for %d s", CONFIG_SLEEP_DURATION_SEC);
  esp_sleep_enable_timer_wakeup((uint64_t)CONFIG_SLEEP_DURATION_SEC *
                                1000000ULL);
  esp_deep_sleep_start();
}
