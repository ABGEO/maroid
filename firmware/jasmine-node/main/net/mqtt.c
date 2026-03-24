#include "mqtt.h"

#include "esp_log.h"
#include "event_wait.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"
#include "mqtt_client.h"

#include <stdatomic.h>
#include <sys/time.h>

#define MQTT_CONNECTED_BIT BIT0
#define MQTT_ALL_PUBLISHED_BIT BIT1
#define MQTT_CONNECT_TIMEOUT_MS 10000
#define MQTT_PUBLISH_TIMEOUT_MS 5000

#define MQTT_READING_BUF_SIZE 100
#define MQTT_TOPIC_BUF_SIZE 100

#if defined(CONFIG_NODE_TYPE_ENVIRONMENT)
#define NODE_TYPE_STR "environment"
#elif defined(CONFIG_NODE_TYPE_PLANT)
#define NODE_TYPE_STR "plant"
#endif

static const char *TAG = "mqtt";

static EventGroupHandle_t s_mqtt_event_group;
static atomic_int s_pending_publishes;

static void mqtt_event_handler(void *handler_args, esp_event_base_t base,
                               int32_t event_id, void *event_data) {
  switch ((esp_mqtt_event_id_t)event_id) {
  case MQTT_EVENT_CONNECTED:
    xEventGroupSetBits(s_mqtt_event_group, MQTT_CONNECTED_BIT);
    ESP_LOGI(TAG, "MQTT_EVENT_CONNECTED");
    break;
  case MQTT_EVENT_DISCONNECTED:
    xEventGroupClearBits(s_mqtt_event_group, MQTT_CONNECTED_BIT);
    ESP_LOGI(TAG, "MQTT_EVENT_DISCONNECTED");
    break;
  case MQTT_EVENT_PUBLISHED:
    if (atomic_fetch_sub(&s_pending_publishes, 1) == 1) {
      xEventGroupSetBits(s_mqtt_event_group, MQTT_ALL_PUBLISHED_BIT);
    }
    break;
  default:
    ESP_LOGI(TAG, "Other MQTT event id:%d", event_id);
    break;
  }
}

esp_err_t mqtt_start(esp_mqtt_client_handle_t *out_client) {
  s_mqtt_event_group = xEventGroupCreate();
  atomic_store(&s_pending_publishes, 0);

  esp_mqtt_client_config_t cfg = {
      .broker.address.uri = CONFIG_MQTT_BROKER_URL,
      .session.protocol_ver = MQTT_PROTOCOL_V_5,
      .credentials.client_id = "jasmine-" NODE_TYPE_STR "-" CONFIG_NODE_ID,
      .credentials.username = CONFIG_MQTT_USERNAME,
      .credentials.authentication.password = CONFIG_MQTT_PASSWORD,
  };

  esp_mqtt_client_handle_t client = esp_mqtt_client_init(&cfg);
  esp_mqtt_client_register_event(client, ESP_EVENT_ANY_ID, mqtt_event_handler,
                                 NULL);
  esp_mqtt_client_start(client);

  esp_err_t err = ESP_ERR_TIMEOUT;
  for (int attempt = 1; attempt <= CONFIG_MQTT_CONNECT_RETRIES; attempt++) {
    ESP_LOGI(TAG, "Waiting for MQTT connection (attempt %d/%d)...", attempt,
             CONFIG_MQTT_CONNECT_RETRIES);
    err = event_wait(s_mqtt_event_group, MQTT_CONNECTED_BIT,
                     pdMS_TO_TICKS(MQTT_CONNECT_TIMEOUT_MS), TAG);
    if (err == ESP_OK) {
      break;
    }
    if (attempt < CONFIG_MQTT_CONNECT_RETRIES) {
      ESP_LOGW(TAG, "MQTT connect attempt %d failed, retrying...", attempt);
    }
  }

  if (err != ESP_OK) {
    ESP_LOGE(TAG, "MQTT connection failed after %d attempts",
             CONFIG_MQTT_CONNECT_RETRIES);
    esp_mqtt_client_destroy(client);
    vEventGroupDelete(s_mqtt_event_group);
    s_mqtt_event_group = NULL;
    return err;
  }

  *out_client = client;
  return ESP_OK;
}

esp_err_t mqtt_wait_published(void) {
  if (atomic_load(&s_pending_publishes) == 0) {
    return ESP_OK;
  }

  return event_wait(s_mqtt_event_group, MQTT_ALL_PUBLISHED_BIT,
                    pdMS_TO_TICKS(MQTT_PUBLISH_TIMEOUT_MS), TAG);
}

void mqtt_stop(esp_mqtt_client_handle_t client) {
  esp_mqtt_client_stop(client);
  esp_mqtt_client_destroy(client);
  vEventGroupDelete(s_mqtt_event_group);
  s_mqtt_event_group = NULL;
}

int mqtt_send_reading(esp_mqtt_client_handle_t client, const char *reading_type,
                      const char *reading) {
  time_t now;
  struct tm timeinfo;
  char reading_buf[MQTT_READING_BUF_SIZE];
  char topic_buf[MQTT_TOPIC_BUF_SIZE];
  char strftime_buf[64];

  time(&now);
  gmtime_r(&now, &timeinfo);
  strftime(strftime_buf, sizeof(strftime_buf), "%Y-%m-%dT%H:%M:%SZ", &timeinfo);

  int n = snprintf(reading_buf, sizeof(reading_buf),
                   "{\"time\":\"%s\",\"value\":%s}", strftime_buf, reading);
  if (n < 0 || (size_t)n >= sizeof(reading_buf)) {
    ESP_LOGE(TAG, "reading_buf truncated for %s", reading_type);
    return -1;
  }

  n = snprintf(topic_buf, sizeof(topic_buf),
               "dev/maroid/jasmine/%s/%s/measurement/%s", NODE_TYPE_STR,
               CONFIG_NODE_ID, reading_type);
  if (n < 0 || (size_t)n >= sizeof(topic_buf)) {
    ESP_LOGE(TAG, "topic_buf truncated for %s", reading_type);
    return -1;
  }

  atomic_fetch_add(&s_pending_publishes, 1);
  int msg_id = esp_mqtt_client_publish(client, topic_buf, reading_buf, 0, 1, 0);
  if (msg_id < 0) {
    atomic_fetch_sub(&s_pending_publishes, 1);
    ESP_LOGE(TAG, "Failed to publish %s reading", reading_type);
  }

  return msg_id;
}
