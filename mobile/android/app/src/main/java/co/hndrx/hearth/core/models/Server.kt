package co.hndrx.hearth.core.models

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class Server(
    @Json(name = "id") val id: String,
    @Json(name = "name") val name: String,
    @Json(name = "icon_url") val iconUrl: String? = null,
    @Json(name = "banner_url") val bannerUrl: String? = null,
    @Json(name = "description") val description: String? = null,
    @Json(name = "owner_id") val ownerId: String,
    @Json(name = "features") val features: List<String> = emptyList(),
    @Json(name = "created_at") val createdAt: String? = null,
)
