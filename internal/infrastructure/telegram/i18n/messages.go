package i18n

import "strconv"

// User binding related messages

// MsgBindMissingCode is shown when user sends /bind without a code
func MsgBindMissingCode(lang Lang) string {
	if lang == EN {
		return "⚠️ <b>Missing Verification Code</b>\n\n" +
			"Usage: <code>/bind &lt;code&gt;</code>\n\n" +
			"Get your verification code from the website settings page"
	}
	return "⚠️ <b>缺少验证码</b>\n\n" +
		"用法：<code>/bind &lt;code&gt;</code>\n\n" +
		"请在网站设置页面获取验证码"
}

// MsgBindSuccess is shown when user binding is successful
func MsgBindSuccess(lang Lang) string {
	if lang == EN {
		return "✅ <b>Binding Successful</b>\n\n" +
			"<blockquote>🔔 You will receive:\n" +
			"• Subscription expiry reminders\n" +
			"• Traffic usage alerts</blockquote>\n\n" +
			"Use /status to view settings, /unbind to unlink"
	}
	return "✅ <b>绑定成功</b>\n\n" +
		"<blockquote>🔔 您将收到：\n" +
		"• 订阅到期提醒\n" +
		"• 流量使用警告</blockquote>\n\n" +
		"使用 /status 查看设置，/unbind 解绑"
}

// MsgBindFailed is shown when user binding fails
func MsgBindFailed(lang Lang) string {
	if lang == EN {
		return "❌ <b>Binding Failed</b>\n\n" +
			"Invalid or expired verification code\n" +
			"Please check and try again"
	}
	return "❌ <b>绑定失败</b>\n\n" +
		"验证码无效或已过期\n" +
		"请检查验证码后重试"
}

// MsgBindRateLimited is shown when user has too many failed attempts
func MsgBindRateLimited(lang Lang) string {
	if lang == EN {
		return "⚠️ <b>Too Many Requests</b>\n\n" +
			"Too many verification attempts\n" +
			"Please try again in 15 minutes"
	}
	return "⚠️ <b>请求过于频繁</b>\n\n" +
		"您的验证尝试次数过多\n" +
		"请15分钟后再试"
}

// User unbind related messages

// MsgUnbindSuccess is shown when user unbinding is successful
func MsgUnbindSuccess(lang Lang) string {
	if lang == EN {
		return "✅ <b>Unlinked</b>\n\n" +
			"🔕 You will no longer receive notifications\n\n" +
			"Use <code>/bind &lt;code&gt;</code> to reconnect anytime"
	}
	return "✅ <b>已解绑</b>\n\n" +
		"🔕 您将不再收到通知\n\n" +
		"随时使用 <code>/bind &lt;code&gt;</code> 重新连接"
}

// MsgUnbindFailed is shown when user unbinding fails
func MsgUnbindFailed(lang Lang) string {
	if lang == EN {
		return "❌ <b>Unbind Failed</b>\n\n" +
			"Operation failed, please try again later"
	}
	return "❌ <b>解绑失败</b>\n\n" +
		"操作失败，请稍后重试"
}

// Status related messages

// MsgStatusError is shown when getting status fails
func MsgStatusError(lang Lang) string {
	if lang == EN {
		return "❌ <b>Error</b>\n\n" +
			"Failed to get status, please try again later"
	}
	return "❌ <b>错误</b>\n\n" +
		"获取状态失败，请稍后重试"
}

// MsgStatusNotConnected is shown when user is not bound
func MsgStatusNotConnected(lang Lang) string {
	if lang == EN {
		return "🔗 <b>Not Connected</b>\n\n" +
			"Your Telegram is not linked to an account\n\n" +
			"<blockquote><b>How to bind:</b>\n" +
			"1️⃣ Go to website settings\n" +
			"2️⃣ Click \"Bind Telegram\"\n" +
			"3️⃣ Copy the verification code\n" +
			"4️⃣ Send <code>/bind &lt;code&gt;</code></blockquote>"
	}
	return "🔗 <b>未连接</b>\n\n" +
		"您的 Telegram 尚未绑定账户\n\n" +
		"<blockquote><b>绑定步骤：</b>\n" +
		"1️⃣ 访问网站设置页面\n" +
		"2️⃣ 点击「绑定 Telegram」\n" +
		"3️⃣ 复制验证码\n" +
		"4️⃣ 发送 <code>/bind &lt;验证码&gt;</code></blockquote>"
}

// MsgStatusConnectedSimple is shown in polling mode when user is bound
func MsgStatusConnectedSimple(lang Lang) string {
	if lang == EN {
		return "📊 <b>Connected</b>\n\n" +
			"Your account is linked\n\n" +
			"Use /unbind to unlink"
	}
	return "📊 <b>已连接</b>\n\n" +
		"您的账户已绑定\n\n" +
		"使用 /unbind 解绑"
}

// BuildStatusConnectedMessage builds a detailed connected status message with notification settings
func BuildStatusConnectedMessage(lang Lang, notifyExpiring bool, expiringDays int, notifyTraffic bool, trafficThreshold int) string {
	if lang == EN {
		return "📊 <b>Notification Settings</b>\n\n" +
			"<b>Status:</b> 🟢 Connected\n\n" +
			"<blockquote><b>Expiry Reminder</b> " + boolToStatus(lang, notifyExpiring) + "\n" +
			"Notify " + strconv.Itoa(expiringDays) + " days before expiry\n\n" +
			"<b>Traffic Alert</b> " + boolToStatus(lang, notifyTraffic) + "\n" +
			"Threshold: " + strconv.Itoa(trafficThreshold) + "%</blockquote>\n\n" +
			"<i>Modify settings on the website</i>"
	}
	return "📊 <b>通知设置</b>\n\n" +
		"<b>状态：</b> 🟢 已连接\n\n" +
		"<blockquote><b>到期提醒</b> " + boolToStatus(lang, notifyExpiring) + "\n" +
		"提前 " + strconv.Itoa(expiringDays) + " 天提醒\n\n" +
		"<b>流量警告</b> " + boolToStatus(lang, notifyTraffic) + "\n" +
		"阈值：" + strconv.Itoa(trafficThreshold) + "%</blockquote>\n\n" +
		"<i>在网站修改设置</i>"
}

// Help messages

// MsgHelpUser is the basic user help message (used in webhook mode)
func MsgHelpUser(lang Lang) string {
	if lang == EN {
		return "🤖 <b>Orris Notification Bot</b>\n\n" +
			"Subscription expiry and traffic usage alerts\n\n" +
			"<blockquote>/bind <code>&lt;code&gt;</code> — Link account\n" +
			"/status — View settings\n" +
			"/unbind — Unlink account\n" +
			"/lang — Switch language\n" +
			"/help — Show help</blockquote>\n\n" +
			"💡 Get your code from website settings, then <code>/bind &lt;code&gt;</code>"
	}
	return "🤖 <b>Orris 通知机器人</b>\n\n" +
		"订阅到期和流量使用提醒服务\n\n" +
		"<blockquote>/bind <code>&lt;code&gt;</code> — 绑定账户\n" +
		"/status — 查看设置\n" +
		"/unbind — 解绑账户\n" +
		"/lang — 切换语言\n" +
		"/help — 显示帮助</blockquote>\n\n" +
		"💡 在网站设置页面获取验证码，然后 <code>/bind &lt;code&gt;</code>"
}

// MsgHelpFull is the full help message with admin commands (used in polling mode)
func MsgHelpFull(lang Lang) string {
	if lang == EN {
		return "🤖 <b>Orris Notification Bot</b>\n\n" +
			"Subscription expiry and traffic usage alerts\n\n" +
			"<blockquote><b>User commands</b>\n" +
			"/bind <code>&lt;code&gt;</code> — Link account\n" +
			"/status — View settings\n" +
			"/unbind — Unlink account\n" +
			"/lang — Switch language\n" +
			"/help — Show help\n\n" +
			"<b>Admin commands</b>\n" +
			"/adminbind <code>&lt;code&gt;</code> — Link admin</blockquote>\n\n" +
			"💡 Get your code from website settings, then <code>/bind &lt;code&gt;</code>"
	}
	return "🤖 <b>Orris 通知机器人</b>\n\n" +
		"订阅到期和流量使用提醒服务\n\n" +
		"<blockquote><b>用户命令</b>\n" +
		"/bind <code>&lt;code&gt;</code> — 绑定账户\n" +
		"/status — 查看设置\n" +
		"/unbind — 解绑账户\n" +
		"/lang — 切换语言\n" +
		"/help — 显示帮助\n\n" +
		"<b>管理员命令</b>\n" +
		"/adminbind <code>&lt;code&gt;</code> — 绑定管理员</blockquote>\n\n" +
		"💡 在网站设置页面获取验证码，然后 <code>/bind &lt;code&gt;</code>"
}

// Admin binding related messages

// MsgAdminFeatureNotEnabled is shown when admin service is not configured
func MsgAdminFeatureNotEnabled(lang Lang) string {
	if lang == EN {
		return "❌ <b>Admin Feature Not Enabled</b>\n\nPlease contact your system administrator"
	}
	return "❌ <b>管理员功能未启用</b>\n\n请联系系统管理员"
}

// MsgAdminFeatureNotEnabledShort is the short version
func MsgAdminFeatureNotEnabledShort(lang Lang) string {
	if lang == EN {
		return "❌ <b>Admin Feature Not Enabled</b>"
	}
	return "❌ <b>管理员功能未启用</b>"
}

// MsgAdminBindMissingCode is shown when admin sends /adminbind without a code
func MsgAdminBindMissingCode(lang Lang) string {
	if lang == EN {
		return "⚠️ <b>Missing Verification Code</b>\n\n" +
			"Usage: <code>/adminbind &lt;code&gt;</code>\n\n" +
			"Get your verification code from the admin panel"
	}
	return "⚠️ <b>缺少验证码</b>\n\n" +
		"用法：<code>/adminbind &lt;code&gt;</code>\n\n" +
		"请在管理后台获取验证码"
}

// MsgAdminBindFailed is shown when admin binding fails (webhook mode)
func MsgAdminBindFailed(lang Lang) string {
	if lang == EN {
		return "❌ <b>Binding Failed</b>\n\n" +
			"<blockquote>Possible reasons:\n" +
			"• Invalid or expired verification code\n" +
			"• You are not an admin\n" +
			"• This Telegram is already bound to another admin</blockquote>"
	}
	return "❌ <b>绑定失败</b>\n\n" +
		"<blockquote>可能原因：\n" +
		"• 验证码无效或已过期\n" +
		"• 您不是管理员账户\n" +
		"• 此 Telegram 已被其他管理员绑定</blockquote>"
}

// MsgAdminBindFailedPolling is shown when admin binding fails (polling mode)
func MsgAdminBindFailedPolling(lang Lang) string {
	if lang == EN {
		return "❌ <b>Binding Failed</b>\n\n" +
			"Invalid code, expired, or you are not an admin\n" +
			"Please check and try again"
	}
	return "❌ <b>绑定失败</b>\n\n" +
		"验证码无效、已过期或您不是管理员\n" +
		"请检查后重试"
}

// MsgAdminBindSuccess is shown when admin binding is successful (webhook mode)
func MsgAdminBindSuccess(lang Lang) string {
	if lang == EN {
		return "✅ <b>Admin Binding Successful</b>\n\n" +
			"<blockquote>🔔 You will receive:\n" +
			"• Node/agent offline alerts\n" +
			"• New user registration\n" +
			"• Payment success\n" +
			"• Daily/weekly business summaries</blockquote>\n\n" +
			"Use /adminstatus to view settings, /adminunbind to unlink"
	}
	return "✅ <b>管理员绑定成功</b>\n\n" +
		"<blockquote>🔔 您将收到：\n" +
		"• 节点/代理离线告警\n" +
		"• 新用户注册通知\n" +
		"• 支付成功通知\n" +
		"• 每日/每周业务摘要</blockquote>\n\n" +
		"使用 /adminstatus 查看设置，/adminunbind 解绑"
}

// MsgAdminBindSuccessPolling is shown when admin binding is successful (polling mode)
func MsgAdminBindSuccessPolling(lang Lang) string {
	if lang == EN {
		return "✅ <b>Admin Binding Successful</b>\n\n" +
			"<blockquote>🔔 You will receive:\n" +
			"• Node offline alerts\n" +
			"• New user registration\n" +
			"• Payment success\n" +
			"• Daily/weekly reports</blockquote>"
	}
	return "✅ <b>管理员绑定成功</b>\n\n" +
		"<blockquote>🔔 您将收到：\n" +
		"• 节点离线告警\n" +
		"• 新用户注册\n" +
		"• 支付成功通知\n" +
		"• 每日/每周报告</blockquote>"
}

// MsgAdminBindRateLimited is shown when admin has too many failed attempts
func MsgAdminBindRateLimited(lang Lang) string {
	if lang == EN {
		return "⚠️ <b>Too Many Requests</b>\n\n" +
			"Too many verification attempts, account temporarily locked\n" +
			"Please try again in 30 minutes"
	}
	return "⚠️ <b>请求过于频繁</b>\n\n" +
		"验证尝试次数过多，账户已临时锁定\n" +
		"请30分钟后再试"
}

// MsgAdminUnbindSuccess is shown when admin unbinding is successful
func MsgAdminUnbindSuccess(lang Lang) string {
	if lang == EN {
		return "✅ <b>Admin Unlinked</b>\n\n🔕 You will no longer receive admin notifications"
	}
	return "✅ <b>管理员已解绑</b>\n\n🔕 您将不再收到管理员通知"
}

// MsgAdminUnbindFailed is shown when admin unbinding fails
func MsgAdminUnbindFailed(lang Lang) string {
	if lang == EN {
		return "❌ <b>Unbind Failed</b>\n\nYou may not have an admin account bound"
	}
	return "❌ <b>解绑失败</b>\n\n您可能未绑定管理员账户"
}

// MsgAdminStatusNotBound is shown when admin is not bound
func MsgAdminStatusNotBound(lang Lang) string {
	if lang == EN {
		return "🔗 <b>Admin Not Bound</b>\n\n" +
			"Use <code>/adminbind &lt;code&gt;</code> to bind admin account"
	}
	return "🔗 <b>未绑定管理员账户</b>\n\n" +
		"使用 <code>/adminbind &lt;code&gt;</code> 绑定管理员账户"
}

// MsgAdminStatusBound is shown when admin is bound
func MsgAdminStatusBound(lang Lang) string {
	if lang == EN {
		return "📊 <b>Admin Notification Status</b>\n\n" +
			"<b>Status:</b> 🟢 Bound\n\n" +
			"<i>Modify notification settings in the admin panel</i>"
	}
	return "📊 <b>管理员通知状态</b>\n\n" +
		"<b>状态：</b> 🟢 已绑定\n\n" +
		"<i>在管理后台修改通知设置</i>"
}

// Callback query related messages

// MsgCallbackInvalidAction is shown when callback data format is invalid
func MsgCallbackInvalidAction(lang Lang) string {
	if lang == EN {
		return "❌ Invalid action"
	}
	return "❌ 无效操作"
}

// MsgCallbackUnknownAction is shown when callback action is not recognized
func MsgCallbackUnknownAction(lang Lang) string {
	if lang == EN {
		return "❌ Unknown action"
	}
	return "❌ 未知操作"
}

// MsgCallbackInvalidRequest is shown when callback request is malformed
func MsgCallbackInvalidRequest(lang Lang) string {
	if lang == EN {
		return "❌ Invalid request"
	}
	return "❌ 无效请求"
}

// MsgCallbackPermissionDenied is shown when user doesn't have permission
func MsgCallbackPermissionDenied(lang Lang) string {
	if lang == EN {
		return "❌ Permission denied"
	}
	return "❌ 无权限操作"
}

// MsgCallbackUnknownResourceType is shown when resource type is not recognized
func MsgCallbackUnknownResourceType(lang Lang) string {
	if lang == EN {
		return "❌ Unknown resource type"
	}
	return "❌ 未知资源类型"
}

// MsgCallbackOperationFailed is shown when operation fails
func MsgCallbackOperationFailed(lang Lang) string {
	if lang == EN {
		return "❌ Operation failed"
	}
	return "❌ 操作失败"
}

// MsgCallbackFeatureNotEnabled is shown when a callback feature is not enabled
func MsgCallbackFeatureNotEnabled(lang Lang) string {
	if lang == EN {
		return "❌ Feature not enabled"
	}
	return "❌ 功能未启用"
}

// MsgCallbackMuteSuccess returns success message for mute (resource name appended)
func MsgCallbackMuteSuccess(lang Lang) string {
	if lang == EN {
		return "✅ Muted "
	}
	return "✅ 已静默此"
}

// MsgCallbackUnmuteSuccess returns success message for unmute (resource name appended)
func MsgCallbackUnmuteSuccess(lang Lang) string {
	if lang == EN {
		return "✅ Unmuted "
	}
	return "✅ 已解除静默"
}

// ResourceName returns the localized display name for a resource type
func ResourceName(lang Lang, resourceType string) string {
	switch resourceType {
	case "agent":
		if lang == EN {
			return "Forward Agent"
		}
		return "转发代理"
	case "node":
		if lang == EN {
			return "Node Agent"
		}
		return "节点代理"
	default:
		return resourceType
	}
}

// Lang command related messages

// MsgLangCurrent shows the current language and usage instructions
func MsgLangCurrent(lang Lang) string {
	if lang == EN {
		return "🌐 <b>Language Settings</b>\n\n" +
			"Current language: <b>English</b>\n\n" +
			"Switch: <code>/lang zh</code> or <code>/lang en</code>"
	}
	return "🌐 <b>语言设置</b>\n\n" +
		"当前语言：<b>中文</b>\n\n" +
		"切换：<code>/lang en</code> 或 <code>/lang zh</code>"
}

// MsgLangSwitched shows the language switch success message (in the NEW language)
func MsgLangSwitched(lang Lang) string {
	if lang == EN {
		return "✅ Language switched to <b>English</b>"
	}
	return "✅ 语言已切换为<b>中文</b>"
}

// MsgLangInvalid shows when an invalid language argument is provided
func MsgLangInvalid(lang Lang) string {
	if lang == EN {
		return "⚠️ <b>Invalid Language</b>\n\n" +
			"Supported: <code>/lang en</code> or <code>/lang zh</code>"
	}
	return "⚠️ <b>无效语言</b>\n\n" +
		"支持：<code>/lang en</code> 或 <code>/lang zh</code>"
}

// MsgLangNotBound shows when an unbound user tries to use /lang
func MsgLangNotBound(lang Lang) string {
	if lang == EN {
		return "🔗 <b>Not Connected</b>\n\n" +
			"Please bind your account first with <code>/bind &lt;code&gt;</code>"
	}
	return "🔗 <b>未连接</b>\n\n" +
		"请先使用 <code>/bind &lt;code&gt;</code> 绑定账户"
}

// MsgDraftProcessing is a neutral draft message shown while processing a command
func MsgDraftProcessing(lang Lang) string {
	if lang == EN {
		return "⏳ Processing..."
	}
	return "⏳ 处理中..."
}

// MsgDraftLoading is a neutral draft message shown while loading data
func MsgDraftLoading(lang Lang) string {
	if lang == EN {
		return "⏳ Loading..."
	}
	return "⏳ 加载中..."
}

// boolToStatus converts a boolean to a status string
func boolToStatus(lang Lang, b bool) string {
	if lang == EN {
		if b {
			return "✅ On"
		}
		return "❌ Off"
	}
	if b {
		return "✅ 开启"
	}
	return "❌ 关闭"
}
