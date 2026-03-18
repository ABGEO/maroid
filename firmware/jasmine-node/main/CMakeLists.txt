set(app_sources
        "main.c"
        "peripherals/adc.c"
        "net/mqtt.c"
        "net/sntp.c"
        "net/wifi.c"
        "storage/nvs.c"
        "sensors/bh1750_task.c"
        "sensors/dht_task.c"
        "sensors/soil_moisture_task.c"
)

idf_component_register(SRCS ${app_sources}
        INCLUDE_DIRS "include"
        REQUIRES "esp_adc mqtt soil-moisture"
        PRIV_REQUIRES "esp_netif nvs_flash esp_wifi")
