package co.hndrx.hearth.app.di

import android.content.Context
import androidx.room.Room
import co.hndrx.hearth.BuildConfig
import co.hndrx.hearth.core.auth.AuthManager
import co.hndrx.hearth.core.gateway.GatewayClient
import co.hndrx.hearth.core.network.AuthInterceptor
import co.hndrx.hearth.core.network.HearthApiService
import co.hndrx.hearth.core.storage.ChannelDao
import co.hndrx.hearth.core.storage.HearthDatabase
import co.hndrx.hearth.core.storage.MessageDao
import co.hndrx.hearth.core.storage.ServerDao
import com.squareup.moshi.Moshi
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object AppModule {

    @Provides
    @Singleton
    fun provideMoshi(): Moshi {
        return Moshi.Builder()
            .build()
    }

    @Provides
    @Singleton
    fun provideAuthManager(
        @ApplicationContext context: Context,
    ): AuthManager {
        return AuthManager(context)
    }

    @Provides
    @Singleton
    fun provideOkHttpClient(authManager: AuthManager): OkHttpClient {
        val loggingInterceptor = HttpLoggingInterceptor().apply {
            level = if (BuildConfig.DEBUG) {
                HttpLoggingInterceptor.Level.BODY
            } else {
                HttpLoggingInterceptor.Level.NONE
            }
        }

        return OkHttpClient.Builder()
            .addInterceptor(AuthInterceptor(authManager))
            .addInterceptor(loggingInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .pingInterval(30, TimeUnit.SECONDS) // Keep WebSocket alive
            .build()
    }

    @Provides
    @Singleton
    fun provideRetrofit(okHttpClient: OkHttpClient, moshi: Moshi): Retrofit {
        return Retrofit.Builder()
            .baseUrl(BuildConfig.BASE_URL)
            .client(okHttpClient)
            .addConverterFactory(MoshiConverterFactory.create(moshi))
            .build()
    }

    @Provides
    @Singleton
    fun provideHearthApiService(
        retrofit: Retrofit,
        authManager: AuthManager,
    ): HearthApiService {
        val service = retrofit.create(HearthApiService::class.java)
        // Wire up the API service reference in AuthManager to avoid circular DI
        authManager.apiService = service
        return service
    }

    @Provides
    @Singleton
    fun provideGatewayClient(
        okHttpClient: OkHttpClient,
        moshi: Moshi,
        authManager: AuthManager,
    ): GatewayClient {
        return GatewayClient(okHttpClient, moshi, authManager)
    }

    @Provides
    @Singleton
    fun provideHearthDatabase(
        @ApplicationContext context: Context,
    ): HearthDatabase {
        return Room.databaseBuilder(
            context,
            HearthDatabase::class.java,
            "hearth_database",
        )
            .fallbackToDestructiveMigration()
            .build()
    }

    @Provides
    fun provideServerDao(database: HearthDatabase): ServerDao = database.serverDao()

    @Provides
    fun provideChannelDao(database: HearthDatabase): ChannelDao = database.channelDao()

    @Provides
    fun provideMessageDao(database: HearthDatabase): MessageDao = database.messageDao()
}
