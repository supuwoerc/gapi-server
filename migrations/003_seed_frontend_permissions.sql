-- 前端菜单权限 (resource_type=2, 控制侧边栏菜单可见性)
INSERT INTO sys_permission (code, name, resource_type, module, resource_path, action, description, created_at, updated_at, deleted_at) VALUES
('dashboard', '仪表盘', 2, '', '/dashboard', 'access', '仪表盘菜单访问权限', NOW(), NOW(), 0),
('tasks', '任务', 2, '', '/tasks', 'access', '任务菜单访问权限', NOW(), NOW(), 0),
('notifications', '通知', 2, '', '/notifications', 'access', '通知菜单访问权限', NOW(), NOW(), 0),
('groups', '群组', 2, '', '/groups', 'access', '群组菜单访问权限', NOW(), NOW(), 0),
('projects', '项目', 2, '', '/projects', 'access', '项目菜单访问权限', NOW(), NOW(), 0),
('documents', '文档', 2, '', '/documents', 'access', '文档菜单访问权限', NOW(), NOW(), 0),
('admin', '管理后台', 2, '', '/admin', 'access', '管理后台菜单访问权限', NOW(), NOW(), 0),
('admin:users', '用户管理', 2, 'admin', '/admin/users', 'access', '用户管理菜单访问权限', NOW(), NOW(), 0),
('admin:roles', '角色管理', 2, 'admin', '/admin/roles', 'access', '角色管理菜单访问权限', NOW(), NOW(), 0),
('admin:permissions', '权限管理', 2, 'admin', '/admin/permissions', 'access', '权限管理菜单访问权限', NOW(), NOW(), 0);

-- 前端路由权限 (resource_type=3, 控制路由可访问性)
INSERT INTO sys_permission (code, name, resource_type, module, resource_path, action, description, created_at, updated_at, deleted_at) VALUES
('route:dashboard', '仪表盘路由', 3, '', '/dashboard', 'access', '仪表盘路由访问权限', NOW(), NOW(), 0),
('route:tasks', '任务路由', 3, '', '/tasks', 'access', '任务路由访问权限', NOW(), NOW(), 0),
('route:notifications', '通知路由', 3, '', '/notifications', 'access', '通知路由访问权限', NOW(), NOW(), 0),
('route:groups', '群组路由', 3, '', '/groups', 'access', '群组路由访问权限', NOW(), NOW(), 0),
('route:projects', '项目路由', 3, '', '/projects', 'access', '项目路由访问权限', NOW(), NOW(), 0),
('route:documents', '文档路由', 3, '', '/documents', 'access', '文档路由访问权限', NOW(), NOW(), 0),
('route:admin', '管理后台路由', 3, '', '/admin', 'access', '管理后台路由访问权限', NOW(), NOW(), 0),
('route:admin:users', '用户管理路由', 3, 'admin', '/admin/users', 'access', '用户管理路由访问权限', NOW(), NOW(), 0),
('route:admin:roles', '角色管理路由', 3, 'admin', '/admin/roles', 'access', '角色管理路由访问权限', NOW(), NOW(), 0),
('route:admin:permissions', '权限管理路由', 3, 'admin', '/admin/permissions', 'access', '权限管理路由访问权限', NOW(), NOW(), 0);

-- 给 admin 角色分配全部前端权限 (menu + route)
INSERT INTO sys_role_permission (role_id, permission_id, effect, created_at, updated_at, deleted_at)
SELECT r.id, p.id, 'allow', NOW(), NOW(), 0
FROM sys_role r, sys_permission p
WHERE r.code = 'admin' AND p.resource_type IN (2, 3);

-- 给 user 角色分配基础前端权限（不含管理后台）
INSERT INTO sys_role_permission (role_id, permission_id, effect, created_at, updated_at, deleted_at)
SELECT r.id, p.id, 'allow', NOW(), NOW(), 0
FROM sys_role r, sys_permission p
WHERE r.code = 'user' AND p.resource_type IN (2, 3) AND p.code NOT LIKE 'admin%' AND p.code NOT LIKE 'route:admin%';