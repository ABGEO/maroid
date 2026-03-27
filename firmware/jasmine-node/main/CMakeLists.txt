set(app_sources
        "main.c"
        "net/mqtt.c"
        "net/sntp.c"
        "net/wifi.c"
        "storage/nvs.c"
        "sensors/sensor_registry.c"
        "sensors/sensor_task.c"
)

if(CONFIG_SENSOR_BH1750_ENABLED)
    list(APPEND app_sources "peripherals/i2c.c" "sensors/bh1750_sensor.c")
endif()

if(CONFIG_SENSOR_DHT_ENABLED)
    list(APPEND app_sources "sensors/dht_sensor.c")
endif()

if(CONFIG_SENSOR_SOIL_MOISTURE_ENABLED OR CONFIG_SENSOR_MQ135_ENABLED)
    list(APPEND app_sources "peripherals/adc.c")
endif()

if(CONFIG_SENSOR_SOIL_MOISTURE_ENABLED)
    list(APPEND app_sources "sensors/soil_moisture_sensor.c")
endif()

if(CONFIG_SENSOR_MQ135_ENABLED)
    list(APPEND app_sources "sensors/mq135_sensor.c")
endif()

idf_component_register(SRCS ${app_sources}
        INCLUDE_DIRS "include" "peripherals" "sensors" "net" "storage"
        REQUIRES "esp_adc mqtt soil-moisture mq135"
        PRIV_REQUIRES "esp_netif nvs_flash esp_wifi")
