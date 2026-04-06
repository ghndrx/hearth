package co.hndrx.hearth.app.navigation

import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChatBubble
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import co.hndrx.hearth.features.auth.LoginScreen
import co.hndrx.hearth.features.auth.RegisterScreen
import co.hndrx.hearth.features.dms.DMListScreen
import co.hndrx.hearth.features.messages.MessageListScreen
import co.hndrx.hearth.features.notifications.NotificationsScreen
import co.hndrx.hearth.features.profile.ProfileScreen
import co.hndrx.hearth.features.servers.ChannelListScreen
import co.hndrx.hearth.features.servers.ServerListScreen

object Routes {
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val MAIN = "main"
    const val SERVERS = "servers"
    const val DMS = "dms"
    const val NOTIFICATIONS = "notifications"
    const val PROFILE = "profile"
    const val CHANNEL_LIST = "channelList/{serverId}"
    const val MESSAGES = "messages/{serverId}/{channelId}"

    fun channelList(serverId: String) = "channelList/$serverId"
    fun messages(serverId: String, channelId: String) = "messages/$serverId/$channelId"
}

sealed class BottomNavItem(
    val route: String,
    val label: String,
    val icon: ImageVector,
) {
    data object Servers : BottomNavItem(Routes.SERVERS, "Servers", Icons.Filled.Dns)
    data object DMs : BottomNavItem(Routes.DMS, "Messages", Icons.Filled.ChatBubble)
    data object Notifications : BottomNavItem(Routes.NOTIFICATIONS, "Alerts", Icons.Filled.Notifications)
    data object Profile : BottomNavItem(Routes.PROFILE, "Profile", Icons.Filled.Person)
}

@Composable
fun HearthNavHost(isAuthenticated: Boolean) {
    val navController = rememberNavController()

    LaunchedEffect(isAuthenticated) {
        if (!isAuthenticated) {
            navController.navigate(Routes.LOGIN) {
                popUpTo(0) { inclusive = true }
            }
        } else {
            navController.navigate(Routes.MAIN) {
                popUpTo(0) { inclusive = true }
            }
        }
    }

    NavHost(
        navController = navController,
        startDestination = if (isAuthenticated) Routes.MAIN else Routes.LOGIN,
    ) {
        composable(Routes.LOGIN) {
            LoginScreen(
                onNavigateToRegister = {
                    navController.navigate(Routes.REGISTER)
                },
                onLoginSuccess = {
                    navController.navigate(Routes.MAIN) {
                        popUpTo(Routes.LOGIN) { inclusive = true }
                    }
                },
            )
        }

        composable(Routes.REGISTER) {
            RegisterScreen(
                onNavigateToLogin = {
                    navController.popBackStack()
                },
                onRegisterSuccess = {
                    navController.navigate(Routes.MAIN) {
                        popUpTo(Routes.LOGIN) { inclusive = true }
                    }
                },
            )
        }

        composable(Routes.MAIN) {
            MainScreen(navController = navController)
        }

        composable(
            route = Routes.CHANNEL_LIST,
            arguments = listOf(navArgument("serverId") { type = NavType.StringType }),
        ) { backStackEntry ->
            val serverId = backStackEntry.arguments?.getString("serverId") ?: return@composable
            ChannelListScreen(
                serverId = serverId,
                onNavigateToMessages = { sId, cId ->
                    navController.navigate(Routes.messages(sId, cId))
                },
                onNavigateBack = {
                    navController.popBackStack()
                },
            )
        }

        composable(
            route = Routes.MESSAGES,
            arguments = listOf(
                navArgument("serverId") { type = NavType.StringType },
                navArgument("channelId") { type = NavType.StringType },
            ),
        ) { backStackEntry ->
            val serverId = backStackEntry.arguments?.getString("serverId") ?: return@composable
            val channelId = backStackEntry.arguments?.getString("channelId") ?: return@composable
            MessageListScreen(
                serverId = serverId,
                channelId = channelId,
                onNavigateBack = {
                    navController.popBackStack()
                },
            )
        }
    }
}

@Composable
fun MainScreen(navController: NavHostController) {
    val bottomNavController = rememberNavController()
    val bottomNavItems = listOf(
        BottomNavItem.Servers,
        BottomNavItem.DMs,
        BottomNavItem.Notifications,
        BottomNavItem.Profile,
    )

    Scaffold(
        bottomBar = {
            NavigationBar {
                val navBackStackEntry by bottomNavController.currentBackStackEntryAsState()
                val currentDestination = navBackStackEntry?.destination

                bottomNavItems.forEach { item ->
                    NavigationBarItem(
                        icon = { Icon(item.icon, contentDescription = item.label) },
                        label = { Text(item.label) },
                        selected = currentDestination?.hierarchy?.any { it.route == item.route } == true,
                        onClick = {
                            bottomNavController.navigate(item.route) {
                                popUpTo(bottomNavController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                    )
                }
            }
        },
    ) { innerPadding ->
        NavHost(
            navController = bottomNavController,
            startDestination = Routes.SERVERS,
            modifier = Modifier.padding(innerPadding),
        ) {
            composable(Routes.SERVERS) {
                ServerListScreen(
                    onNavigateToChannelList = { serverId ->
                        navController.navigate(Routes.channelList(serverId))
                    },
                )
            }
            composable(Routes.DMS) {
                DMListScreen(
                    onNavigateToDM = { channelId ->
                        // DMs use a pseudo server id
                        navController.navigate(Routes.messages("@me", channelId))
                    },
                )
            }
            composable(Routes.NOTIFICATIONS) {
                NotificationsScreen()
            }
            composable(Routes.PROFILE) {
                ProfileScreen()
            }
        }
    }
}
